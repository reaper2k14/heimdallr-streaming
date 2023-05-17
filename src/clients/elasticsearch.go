package clients

import (
	"errors"
	"os"

	"github.com/elastic/go-elasticsearch/v6"
)

var elasticsearchClient *elasticsearch.Client = nil
var errorEmptyHostURL error = errors.New("the elasticsearch host url is empty")

func GetElasticsearch() (*elasticsearch.Client, error) {
	if elasticsearchClient == nil {
		elasticsearchUrl := os.Getenv("ELASTICSEARCH_URL")
		if len(elasticsearchUrl) == 0 || elasticsearchUrl == "" {
			return nil, errorEmptyHostURL
		}

		cfg := elasticsearch.Config{
			Addresses: []string{
				elasticsearchUrl,
			},
		}
		client, err := elasticsearch.NewClient(cfg)
		if err != nil {
			return nil, err
		}
		elasticsearchClient = client
	}
	return elasticsearchClient, nil
}
