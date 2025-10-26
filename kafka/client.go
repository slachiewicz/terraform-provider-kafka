package kafka

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

type TopicMissingError struct {
	msg string
}

func (e TopicMissingError) Error() string { return e.msg }

type void struct{}

var member void

type aclCache struct {
	acls  []*sarama.ResourceAcls
	mutex sync.RWMutex
	valid bool
}

type aclDeletionQueue struct {
	filters   []*sarama.AclFilter
	after     time.Duration
	timer     *time.Timer
	mutex     sync.Mutex
	waitChans []chan error
}

type aclCreationQueue struct {
	creations []*sarama.AclCreation
	after     time.Duration
	timer     *time.Timer
	mutex     sync.Mutex
	waitChans []chan error
}

type Client struct {
	client      sarama.Client
	kafkaConfig *sarama.Config
	config      *Config
	topics      map[string]void
	topicsMutex sync.RWMutex
	aclCache
	aclDeletionQueue
	aclCreationQueue
}

func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("cannot create client without kafka config")
	}

	log.Printf("[TRACE] configuring bootstrap_servers %v", config.copyWithMaskedSensitiveValues())
	if config.BootstrapServers == nil {
		return nil, fmt.Errorf("no bootstrap_servers provided")
	}

	bootstrapServers := *(config.BootstrapServers)
	if bootstrapServers == nil {
		return nil, fmt.Errorf("no bootstrap_servers provided")
	}

	log.Printf("[INFO] configuring kafka client with %v", config.copyWithMaskedSensitiveValues())

	kc, err := config.newKafkaConfig()
	if err != nil {
		log.Printf("[ERROR] Error creating kafka client %v", err)
		return nil, err
	}

	c, err := sarama.NewClient(bootstrapServers, kc)
	if err != nil {
		log.Printf("[ERROR] Error connecting to kafka %s", err)
		return nil, err
	}

	client := &Client{
		client:      c,
		config:      config,
		kafkaConfig: kc,
		aclDeletionQueue: aclDeletionQueue{
			after: time.Millisecond * 500,
		},
		aclCreationQueue: aclCreationQueue{
			after: time.Millisecond * 500,
		},
	}

	err = client.extractTopics()

	return client, err
}

func (c *Client) SaramaClient() sarama.Client {
	return c.client
}

func (c *Client) extractTopics() error {
	topics, err := c.client.Topics()
	if err != nil {
		log.Printf("[ERROR] Error getting topics %s from Kafka", err)
		return err
	}
	log.Printf("[DEBUG] Got %d topics from Kafka", len(topics))
	c.topicsMutex.Lock()
	c.topics = make(map[string]void)
	for _, t := range topics {
		c.topics[t] = member
	}
	c.topicsMutex.Unlock()
	return nil
}

func (c *Client) DeleteTopic(t string) error {
	broker, err := c.client.Controller()
	if err != nil {
		return err
	}

	timeout := time.Duration(c.config.Timeout) * time.Second
	req := sarama.NewDeleteTopicsRequest(c.kafkaConfig.Version, []string{t}, timeout)

	res, err := broker.DeleteTopics(req)
	if err == nil {
		for k, e := range res.TopicErrorCodes {
			if e != sarama.ErrNoError {
				return fmt.Errorf("%s : %s", k, e)
			}
		}
	} else {
		log.Printf("[ERROR] Error deleting topic %s from Kafka: %s", t, err)
		return err
	}

	log.Printf("[INFO] Deleted topic %s from Kafka", t)

	return nil
}

func (c *Client) UpdateTopic(topic Topic) error {
	broker, err := c.client.Controller()
	if err != nil {
		return err
	}

	r := &sarama.AlterConfigsRequest{
		Version:      1, // Version 1 is supported since Kafka 2.0.0
		Resources:    configToResources(topic, c.config),
		ValidateOnly: false,
	}

	res, err := broker.AlterConfigs(r)
	if err != nil {
		return err
	}

	for _, e := range res.Resources {
		if e.ErrorCode != int16(sarama.ErrNoError) {
			return errors.New(e.ErrorMsg)
		}
	}

	return nil
}

func (c *Client) CreateTopic(t Topic) error {
	broker, err := c.client.Controller()
	if err != nil {
		return err
	}

	timeout := time.Duration(c.config.Timeout) * time.Second
	log.Printf("[TRACE] Timeout is %v ", timeout)

	topicDetails := map[string]*sarama.TopicDetail{
		t.Name: {
			NumPartitions:     t.Partitions,
			ReplicationFactor: t.ReplicationFactor,
			ConfigEntries:     t.Config,
		},
	}
	req := sarama.NewCreateTopicsRequest(c.kafkaConfig.Version, topicDetails, timeout, false)

	res, err := broker.CreateTopics(req)

	if err == nil {
		for _, e := range res.TopicErrors {
			if e.Err != sarama.ErrNoError {
				return fmt.Errorf("%s", e.Err)
			}
		}
		log.Printf("[INFO] Created topic %s in Kafka", t.Name)
	}

	return err
}

func (c *Client) AddPartitions(t Topic) error {
	broker, err := c.client.Controller()
	if err != nil {
		return err
	}

	timeout := time.Duration(c.config.Timeout) * time.Second
	tp := map[string]*sarama.TopicPartition{
		t.Name: &sarama.TopicPartition{
			Count: t.Partitions,
		},
	}

	req := &sarama.CreatePartitionsRequest{
		Version:         1, // Version 1 is supported since Kafka 2.0.0
		TopicPartitions: tp,
		Timeout:         timeout,
		ValidateOnly:    false,
	}

	log.Printf("[INFO] Adding partitions to %s in Kafka", t.Name)
	res, err := broker.CreatePartitions(req)
	if err == nil {
		for _, e := range res.TopicPartitionErrors {
			if e.Err != sarama.ErrNoError {
				return fmt.Errorf("%s", e.Err)
			}
		}
		log.Printf("[INFO] Added partitions to %s in Kafka", t.Name)
	}

	return err
}

func (c *Client) CanAlterReplicationFactor() bool {
	// AlterPartitionReassignments requires Kafka 2.4.0 or higher
	// With Sarama 1.46+, version negotiation is automatic, so we just check
	// if we can create a ClusterAdmin (which is required for the operation)
	admin, err := sarama.NewClusterAdminFromClient(c.client)
	if err != nil {
		return false
	}
	defer admin.Close()

	// The feature is available if we can create the admin client
	// Sarama will handle API version negotiation internally
	return true
}

func (c *Client) AlterReplicationFactor(t Topic) error {
	log.Printf("[DEBUG] Refreshing metadata for topic '%s'", t.Name)
	if err := c.client.RefreshMetadata(t.Name); err != nil {
		return err
	}

	admin, err := sarama.NewClusterAdminFromClient(c.client)
	if err != nil {
		return err
	}

	assignment, err := c.buildAssignment(t)
	if err != nil {
		return err
	}

	return admin.AlterPartitionReassignments(t.Name, *assignment)
}

func (c *Client) buildAssignment(t Topic) (*[][]int32, error) {
	partitions, err := c.client.Partitions(t.Name)
	if err != nil {
		return nil, err
	}

	allReplicas := c.allReplicas()
	newRF := t.ReplicationFactor

	assignment := make([][]int32, len(partitions))
	for _, p := range partitions {
		oldReplicas, err := c.client.Replicas(t.Name, p)
		if err != nil {
			return &assignment, err
		}

		oldRF := int16(len(oldReplicas))
		deltaRF := newRF - oldRF
		newReplicas, err := buildNewReplicas(allReplicas, &oldReplicas, deltaRF)
		if err != nil {
			return &assignment, err
		}

		assignment[p] = *newReplicas
	}

	return &assignment, nil
}

func (c *Client) allReplicas() *[]int32 {
	brokers := c.client.Brokers()
	replicas := make([]int32, 0, len(brokers))

	for _, b := range brokers {
		id := b.ID()
		if id != -1 {
			replicas = append(replicas, id)
		}
	}

	return &replicas
}

func buildNewReplicas(allReplicas *[]int32, usedReplicas *[]int32, deltaRF int16) (*[]int32, error) {
	usedCount := int16(len(*usedReplicas))

	if deltaRF == 0 {
		return usedReplicas, nil
	} else if deltaRF < 0 {
		end := usedCount + deltaRF
		if end < 1 {
			return nil, errors.New("dropping too many replicas")
		}

		head := (*usedReplicas)[:end]
		return &head, nil
	} else {
		extraCount := int16(len(*allReplicas)) - usedCount
		if extraCount < deltaRF {
			return nil, errors.New("not enough brokers")
		}

		unusedReplicas := *findUnusedReplicas(allReplicas, usedReplicas, extraCount)
		newReplicas := *usedReplicas
		for i := int16(0); i < deltaRF; i++ {
			j := rand.Intn(len(unusedReplicas))
			newReplicas = append(newReplicas, unusedReplicas[j])
			unusedReplicas[j] = unusedReplicas[len(unusedReplicas)-1]
			unusedReplicas = unusedReplicas[:len(unusedReplicas)-1]
		}

		return &newReplicas, nil
	}
}

func findUnusedReplicas(allReplicas *[]int32, usedReplicas *[]int32, extraCount int16) *[]int32 {
	usedMap := make(map[int32]bool, len(*usedReplicas))
	for _, r := range *usedReplicas {
		usedMap[r] = true
	}

	unusedReplicas := make([]int32, 0, extraCount)
	for _, r := range *allReplicas {
		_, exists := usedMap[r]
		if !exists {
			unusedReplicas = append(unusedReplicas, r)
		}
	}

	return &unusedReplicas
}

func (c *Client) IsReplicationFactorUpdating(topic string) (bool, error) {
	log.Printf("[DEBUG] Refreshing metadata for topic '%s'", topic)
	if err := c.client.RefreshMetadata(topic); err != nil {
		return false, err
	}

	partitions, err := c.client.Partitions(topic)
	if err != nil {
		return false, err
	}

	admin, err := sarama.NewClusterAdminFromClient(c.client)
	if err != nil {
		return false, err
	}

	statusMap, err := admin.ListPartitionReassignments(topic, partitions)
	if err != nil {
		return false, err
	}

	for _, status := range statusMap[topic] {
		if isPartitionRFChanging(status) {
			return true, nil
		}
	}

	return false, nil
}

func isPartitionRFChanging(status *sarama.PartitionReplicaReassignmentsStatus) bool {
	return len(status.AddingReplicas) != 0 || len(status.RemovingReplicas) != 0
}

func (client *Client) ReadTopic(name string, refreshMetadata bool) (Topic, error) {
	c := client.client
	log.Printf("[INFO] 👋 reading topic '%s' from Kafka: %v", name, refreshMetadata)

	topic := Topic{
		Name: name,
	}

	if refreshMetadata {
		log.Printf("[DEBUG] Refreshing metadata for topic '%s'", name)
		err := c.RefreshMetadata(name)

		if err == sarama.ErrUnknownTopicOrPartition {
			err := TopicMissingError{msg: fmt.Sprintf("%s could not be found", name)}
			return topic, err
		}
		if err != nil {
			log.Printf("[ERROR] Error refreshing topic '%s' metadata %s", name, err)
			return topic, err
		}
		err = client.extractTopics()
		if err != nil {
			return topic, err
		}
	} else {
		log.Printf("[DEBUG] skipping metadata refresh for topic '%s'", name)
	}

	client.topicsMutex.RLock()
	_, topicExists := client.topics[name]
	client.topicsMutex.RUnlock()
	if topicExists {
		log.Printf("[DEBUG] Found %s from Kafka", name)
		p, err := c.Partitions(name)
		if err == nil {
			partitionCount := int32(len(p))
			log.Printf("[DEBUG] [%s] %d Partitions Found: %v from Kafka", name, partitionCount, p)
			topic.Partitions = partitionCount

			r, err := ReplicaCount(c, name, p)
			if err != nil {
				return topic, err
			}

			log.Printf("[DEBUG] [%s] ReplicationFactor %d from Kafka", name, r)
			topic.ReplicationFactor = int16(r)

			configToSave, err := client.topicConfig(name)
			if err != nil {
				log.Printf("[ERROR] [%s] Could not get config for topic %s", name, err)
				return topic, err
			}

			log.Printf("[TRACE] [%s] Config %v from Kafka", name, strPtrMapToStrMap(configToSave))
			topic.Config = configToSave
			return topic, nil
		}
	}

	err := TopicMissingError{msg: fmt.Sprintf("%s could not be found", name)}
	return topic, err
}

// topicConfig retrives the non-default config map for a topic
func (c *Client) topicConfig(topic string) (map[string]*string, error) {
	conf := map[string]*string{}
	request := &sarama.DescribeConfigsRequest{
		Version: 2, // Version 2 is supported since Kafka 2.0.0 and includes synonym support
		Resources: []*sarama.ConfigResource{
			{
				Type: sarama.TopicResource,
				Name: topic,
			},
		},
	}

	broker, err := c.client.Controller()
	if err != nil {
		return conf, err
	}

	cr, err := broker.DescribeConfigs(request)
	if err != nil {
		return conf, err
	}

	if len(cr.Resources) > 0 && len(cr.Resources[0].Configs) > 0 {
		for _, tConf := range cr.Resources[0].Configs {
			v := tConf.Value
			log.Printf("[TRACE] [%s] %s: %v. Default %v, Source %v, Version %d", topic, tConf.Name, v, tConf.Default, tConf.Source, cr.Version)

			for _, s := range tConf.Synonyms {
				log.Printf("[TRACE] Syonyms: %v", s)
			}

			if tConf.Name == "segment.bytes" && c.config.isAWSMSKServerless() {
				// Remove segment.bytes in AWS MSK Serverless response to prevent perpetual planning
				log.Printf("[TRACE] [%s] Using AWS MSK Serverless. Skipping segment.bytes config", topic)
				continue
			}

			if isDefault(tConf, int(cr.Version)) {
				continue
			}
			conf[tConf.Name] = &v
		}
	}
	return conf, nil
}

func (c *Client) getKafkaTopics() ([]Topic, error) {
	topics, err := c.client.Topics()
	if err != nil {
		return nil, err
	}
	topicList := make([]Topic, len(topics))
	for i := range topicList {
		topicList[i], err = c.ReadTopic(topics[i], true)
		if err != nil {
			return nil, err
		}
	}
	return topicList, nil
}
