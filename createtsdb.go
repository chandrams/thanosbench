package main
import (
	"flag"
	"fmt"
	"time"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"strconv"
	"strings"
)

// Global variables
var (
	data         string
	defaultProfile      string
	image        string
	orgCount     int
	clusterCount int
	workerCount  int
	maxTime      string
	store1Data string

)


func execCmd(cmd string, flags ...string) error {
	c := exec.Command(cmd, flags...)
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s %v\nError: %v\nOutput: %s", cmd, flags, err, string(output))
	}
	return nil
}

func createData() error {
	profile := os.Getenv("BLOCK_PROFILE")
	if profile == "" {
		profile = defaultProfile
	}

	fmt.Println("Re-creating data (can take minutes)...")

	commands := make(chan string, orgCount*clusterCount) // Buffered channel

	// Generate commands dynamically
	go func() {
		fmt.Println("Generating commands...")
		for org := 1; org <= orgCount; org++ {
			orgID := fmt.Sprintf("org-%d", org)
			for cluster := 1; cluster <= clusterCount; cluster++ {
				clusterID := fmt.Sprintf("eu-%d-%d", org, cluster)

				commandStr := fmt.Sprintf(
					`mkdir -p %s && docker run --rm -i %s block plan -p %s --labels 'cluster_id="%s"' --labels 'org_id="%s"' --max-time=%s |
                    docker run --rm -v %s/:/shared:z -i %s block gen --output.dir /shared`,
					store1Data, image, profile, clusterID, orgID, maxTime, store1Data, image)
				fmt.Printf("cmd - %s", commandStr)

				commands <- commandStr // Send command to the queue
			}
		}
		close(commands) // Close channel after all commands are added
	}()


	// Use WaitGroup and a channel to handle concurrency
	var wg sync.WaitGroup

	// Start fixed number of workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for command := range commands { // Workers process jobs from the queue
				if err := execCmd("sh", "-c", command); err != nil {
					fmt.Printf("Error executing command: %v\n", err)
				}
			}
		}()
	}

	wg.Wait() // Wait for all workers to finish
	fmt.Println("Create data completed!")
	return nil
}

func main() {
	flag.StringVar(&defaultProfile, "profile", "kruize-15d-1k", "Default kruize profile")
	flag.StringVar(&image, "image", "quay.io/chandra25ms/thanosbench:kruize1", "Thanosbench container image")
	flag.IntVar(&orgCount, "orgs", 2, "Number of organizations")
	flag.IntVar(&clusterCount, "clusters", 2, "Number of clusters")
	flag.IntVar(&workerCount, "workers", 3, "Number of workers")
	flag.StringVar(&maxTime, "maxtime", "", "Max Time for tsdb blocks format (e.g., 2025-04-10T00:00:00Z)")

	
	flag.Parse()
	if maxTime == "" {
		maxTime = time.Now().UTC().Format(time.RFC3339)
	}

	stime := strings.ReplaceAll(maxTime, ":", "")
        fmt.Println("time for dir:", stime)

        data := "data" + "_org" + strconv.Itoa(orgCount) + "_cluster" + strconv.Itoa(clusterCount) + "_" + defaultProfile + "_" + stime

	fmt.Println("Data:", data)
	fmt.Println("Profile:", defaultProfile)
	fmt.Println("Image:", image)
	fmt.Println("Orgs:", orgCount)
	fmt.Println("Clusters:", clusterCount)
	fmt.Println("Workers:", workerCount)
	fmt.Println("maxTime:", maxTime)

	store1Data = func() string { a, _ := filepath.Abs(filepath.Join(data, "store1")); return a }()

	start := time.Now()

	if err := createData(); err != nil {
		fmt.Printf("Error creating data: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(start)

	fmt.Println("Execution time:", elapsed)
}

