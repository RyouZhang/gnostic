package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	metrics "github.com/googleapis/gnostic/metrics"
	"github.com/googleapis/gnostic/metrics/vocabulary"
	"github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/interpret/unordered"
)

var (
	proto      = flag.String("proto", "", "path to the Protocol Buffer file")
	path       = flag.String("path", "", "path to directory containing proto files")
	debug      = flag.Bool("debug", false, "debug flag to output more parsing process detail")
	permissive = flag.Bool("permissive", true, "permissive flag to allow the permissive parsing rather than the just documented spec")
)

// Vocabulary ...
type Vocabulary struct {
	Schemas    map[string]int
	Operations map[string]int
	Parameters map[string]int
	Properties map[string]int
}

// NewVocabulary ...
func NewVocabulary() *Vocabulary {
	return &Vocabulary{
		Schemas:    make(map[string]int),
		Operations: make(map[string]int),
		Parameters: make(map[string]int),
		Properties: make(map[string]int),
	}
}

func main() {
	err := run()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(-1)
	}
	os.Exit(0)
}

func run() error {
	flag.Parse()

	if *proto != "" {
		v := NewVocabulary()
		err := v.fillVocabularyFromProto(*proto)
		if err != nil {
			return err
		}
		vocabulary.WriteCSV(&metrics.Vocabulary{
			Properties: fillProtoStructure(v.Properties),
			Schemas:    fillProtoStructure(v.Schemas),
			Operations: fillProtoStructure(v.Operations),
			Parameters: fillProtoStructure(v.Parameters),
		}, "vocabulary.csv")
	} else if *path != "" {
		v := NewVocabulary()
		err := v.fillVocabularyFromPath(*path)
		if err != nil {
			return err
		}
		vocabulary.WriteCSV(&metrics.Vocabulary{
			Properties: fillProtoStructure(v.Properties),
			Schemas:    fillProtoStructure(v.Schemas),
			Operations: fillProtoStructure(v.Operations),
			Parameters: fillProtoStructure(v.Parameters),
		}, "vocabulary.csv")
	} else {
		return fmt.Errorf("please specify an input with --proto")
	}
	return nil
}

func (vocab *Vocabulary) fillVocabularyFromPath(path string) error {
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, ".proto") {
				err := vocab.fillVocabularyFromProto(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
	return err
}

func (vocab *Vocabulary) fillVocabularyFromProto(filename string) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	p, err := protoparser.Parse(
		reader,
		protoparser.WithDebug(*debug),
		protoparser.WithPermissive(*permissive),
		protoparser.WithFilename(filepath.Base(*proto)),
	)
	if err != nil {
		return err
	}
	v, err := protoparser.UnorderedInterpret(p)
	if err != nil {
		return err
	}
	log.Printf("%+v", v)

	for _, m := range v.ProtoBody.Messages {
		vocab.fillVocabularyFromMessage(m)
	}
	for _, s := range v.ProtoBody.Services {
		vocab.fillVocabularyFromService(s)
	}
	return nil
}

func (vocab *Vocabulary) fillVocabularyFromMessage(m *unordered.Message) {
	vocab.Schemas[m.MessageName]++
	for _, f := range m.MessageBody.Fields {
		vocab.Properties[f.FieldName]++
	}
}

func (vocab *Vocabulary) fillVocabularyFromService(m *unordered.Service) {

	for _, op := range m.ServiceBody.RPCs {
		vocab.Operations[op.RPCName]++
	}
}

// fillProtoStructure adds data to the Word Count structure.
// The Word Count structure can then be added to the Vocabulary protocol buffer.
func fillProtoStructure(m map[string]int) []*metrics.WordCount {
	keyNames := make([]string, 0, len(m))
	for key := range m {
		keyNames = append(keyNames, key)
	}
	sort.Strings(keyNames)

	counts := make([]*metrics.WordCount, 0)
	for _, k := range keyNames {
		temp := &metrics.WordCount{
			Word:  k,
			Count: int32(m[k]),
		}
		counts = append(counts, temp)
	}
	return counts
}