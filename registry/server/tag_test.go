package server

import (
	"testing"

	pb "github.com/DataDog/kafka-kit/registry/protos"
)

func TestTagSetFromObject(t *testing.T) {
	topic := &pb.Topic{
		Name:        "test",
		Partitions:  32,
		Replication: 3,
	}

	ts := tagSetFromObject(topic)
	if len(ts) != 3 {
		t.Errorf("Expected tagSet len 3, got %d", len(ts))
	}

	expected := map[string]string{
		"name":        "test",
		"partitions":  "32",
		"replication": "3",
	}

	for k, v := range expected {
		if ts[k] != v {
			t.Errorf("Expected value %s for key %s, got %s", v, k, ts[k])
		}
	}
}

func TestMatchAll(t *testing.T) {
	ts := tagSet{
		"k1": "v1",
		"k2": "v2",
	}

	tSets := map[int]tagSet{
		1: tagSet{"k1": "v1"},
		2: tagSet{"k1": "v1", "k2": "v2"},
		3: tagSet{"k1": "v1", "k2": "v2", "unrelated": "v3"},
	}

	tests := map[int]bool{
		1: false,
		2: true,
		3: true,
	}

	for i, expected := range tests {
		ts2 := tSets[i]
		if ok := ts2.matchAll(ts); ok != expected {
			t.Errorf("Expected tagSet %v matchAll=%v with %v", ts2, expected, ts)
		}
	}
}

func TestTagSet(t *testing.T) {
	tags := tags{"k1:v1", "k2:v2", "k3:v3"}

	ts, err := tags.tagSet()
	if err != nil {
		t.Error("Unexpected error")
	}

	expected := tagSet{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}

	if len(ts) != len(expected) {
		t.Error("Unexpected tagSet size")
	}

	for k, v := range expected {
		if ts[k] != v {
			t.Errorf("Expected value %s for key %s, got %s", v, k, ts[k])
		}
	}
}

func TestFilterTopics(t *testing.T) {
	th := NewTagHandler()

	topics := TopicSet{
		"test_topic1": &pb.Topic{
			Name:        "test_topic1",
			Partitions:  32,
			Replication: 3,
		},
		"test_topic2": &pb.Topic{
			Name:        "test_topic2",
			Partitions:  32,
			Replication: 2,
		},
		"test_topic3": &pb.Topic{
			Name:        "test_topic3",
			Partitions:  16,
			Replication: 2,
		},
	}

	expected := map[int][]string{
		0: []string{"test_topic1", "test_topic2", "test_topic3"},
		1: []string{"test_topic1", "test_topic2"},
		2: []string{"test_topic2"},
	}

	tests := []tags{
		tags{},
		tags{"partitions:32"},
		tags{"partitions:32", "replication:2"},
	}

	for i, tags := range tests {
		filtered, err := th.FilterTopics(topics, tags)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if !stringsEqual(filtered.Names(), expected[i]) {
			t.Errorf("Expected %s, got %s", expected[i], filtered.Names())
		}
	}
}

func stringsEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

func TestFilterBrokers(t *testing.T) {
	th := NewTagHandler()

	brokers := BrokerSet{
		1001: &pb.Broker{
			Id:   1001,
			Rack: "rack1",
		},
		1002: &pb.Broker{
			Id:   1002,
			Rack: "rack2",
		},
		1003: &pb.Broker{
			Id:   1003,
			Rack: "rack1",
		},
	}

	expected := map[int][]uint32{
		0: []uint32{1001, 1002, 1003},
		1: []uint32{1001, 1003},
		2: []uint32{1003},
	}

	tests := []tags{
		tags{},
		tags{"rack:rack1"},
		tags{"rack:rack1", "id:1003"},
	}

	for i, tags := range tests {
		filtered, err := th.FilterBrokers(brokers, tags)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}

		if !intsEqual(filtered.IDs(), expected[i]) {
			t.Errorf("Expected %v, got %v", expected[i], filtered.IDs())
		}
	}
}
func intsEqual(s1, s2 []uint32) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

func TestRestrictedFields(t *testing.T) {
	rs := restrictedFields()

	topicExpected := map[string]struct{}{
		"tags":        struct{}{},
		"name":        struct{}{},
		"partitions":  struct{}{},
		"replication": struct{}{},
	}

	brokerExpected := map[string]struct{}{
		"id":                          struct{}{},
		"rack":                        struct{}{},
		"jmxport":                     struct{}{},
		"timestamp":                   struct{}{},
		"tags":                        struct{}{},
		"listenersecurityprotocolmap": struct{}{},
		"endpoints":                   struct{}{},
		"host":                        struct{}{},
		"port":                        struct{}{},
		"version":                     struct{}{},
	}

	for i, expected := range []map[string]struct{}{topicExpected, brokerExpected} {
		var typeTest string

		switch i {
		case 0:
			typeTest = "topic"
		case 1:
			typeTest = "broker"
		}

		have := rs[typeTest]

		// Compare expected lengths.
		if len(expected) != len(have) {
			t.Errorf("Expected %d fields for %s, got %d",
				len(expected), typeTest, len(have))
		}

		// Compare fields.
		for f := range expected {
			if _, exist := have[f]; !exist {
				t.Errorf("Expected %s to have field %s", typeTest, f)
			}
		}
	}
}
