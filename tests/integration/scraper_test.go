package integration

import (
	"os"
	"time"

	"github.com/lisanmuaddib/agent-go/pkg/masa/masatwitter"
	"github.com/lisanmuaddib/agent-go/pkg/masa/scraper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

const (
	apiEndpoint      = "http://localhost:8080/api/v1/data/twitter/tweets/recent"
	configPath       = "../../pkg/masa/scraper/list.json"
	requestTimeout   = 120 * time.Second
	tweetsPerRequest = 100
	waitTimeout      = 600 * time.Second // Timeout for waiting for all workers
)

var _ = Describe("Scraper Integration", func() {
	var (
		scr    *scraper.Scraper
		logger *logrus.Logger
		client *masatwitter.Client
		config *scraper.ScraperConfig
	)

	BeforeEach(func() {
		// Skip if not running integration tests
		if os.Getenv("INTEGRATION_TESTS") != "true" {
			Skip("Skipping integration test")
		}

		// Setup logger
		logger = logrus.New()
		logger.SetLevel(logrus.DebugLevel)

		// Initialize Masa Twitter client
		clientConfig := &masatwitter.Config{
			APIEndpoint:      apiEndpoint,
			RequestTimeout:   requestTimeout,
			TweetsPerRequest: tweetsPerRequest,
			Logger:           logger,
		}
		client = masatwitter.NewClient(clientConfig)

		// Create scraper instance
		scr = scraper.NewScraper(client, logger)

		// Load test configuration
		var err error
		config, err = scraper.LoadConfig(configPath)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when scraping tweets", func() {
		It("should successfully scrape tweets", func() {
			// Calculate expected number of tasks based on date ranges and number of queries
			expectedTasks := len(config.Tasks)

			// Process tasks synchronously
			err := scr.ProcessTasks(config)
			Expect(err).NotTo(HaveOccurred())

			// Verify final status
			finalStatus := scr.GetStatus()
			Expect(finalStatus.TotalTasks).To(Equal(expectedTasks))
			Expect(finalStatus.CompletedTasks + finalStatus.FailedTasks).To(Equal(finalStatus.TotalTasks))
			Expect(finalStatus.FailedTasks).To(Equal(0))
			Expect(finalStatus.RetryingTasks).To(Equal(0))
		})
	})
})
