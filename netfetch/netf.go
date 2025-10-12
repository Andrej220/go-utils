package netfetch

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func showHelp() {
	fmt.Printf("Usage: %s [OPTIONS] URL\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nExamples:")
	fmt.Printf("  %s -v https://httpbin.org/get\n", os.Args[0])
	fmt.Printf("  %s -I https://httpbin.org/get\n", os.Args[0])
	fmt.Printf("  %s -s -X POST https://httpbin.org/post\n", os.Args[0])
}

func main() {
	// CLI flags
	verbose := flag.Bool("v", false, "Verbose mode (show headers)")
	silent := flag.Bool("s", false, "Silent mode (no progress/output)")
	head := flag.Bool("I", false, "Show headers only (HEAD request)")
	headLong := flag.Bool("head", false, "Show headers only (HEAD request)")
	timeout := flag.Int("timeout", 30, "Timeout in seconds")
	method := flag.String("X", "GET", "HTTP method (GET, POST, etc.)")
	help := flag.Bool("h", false, "Show help")

	flag.Parse()

	// Show help if requested or no URL provided
	if *help || len(flag.Args()) == 0 {
		showHelp()
		return
	}

	url := flag.Arg(0)

	// Validate URL format
	if !isValidURL(url) {
		fmt.Fprintf(os.Stderr, "Error: Invalid URL format: %s\n", url)
		os.Exit(1)
	}

	// Override method to HEAD if -I or --head is used
	if *head || *headLong {
		*method = "HEAD"
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if *verbose && len(via) > 0 {
				fmt.Fprintf(os.Stderr, "Redirect: %s -> %s\n", via[len(via)-1].URL, req.URL)
			}
			return nil
		},
	}

	// Create request
	req, err := http.NewRequest(*method, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	// Make request
	if *verbose && !*silent {
		fmt.Fprintf(os.Stderr, "> %s %s HTTP/1.1\n", *method, url)
		fmt.Fprintf(os.Stderr, "> Host: %s\n", req.URL.Host)
		fmt.Fprintf(os.Stderr, "> User-Agent: go-curl/1.0\n")
		fmt.Fprintf(os.Stderr, "> \n")
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Output based on flags
	if (*head || *headLong) && !*silent {
		// HEAD mode: show only status and headers
		fmt.Printf("HTTP/%d.%d %d %s\n",
			resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)

		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Printf("%s: %s\n", key, value)
			}
		}
		os.Exit(0)
	}

	if *verbose && !*silent {
		// Verbose output (like curl -v)
		fmt.Fprintf(os.Stderr, "< HTTP/%d.%d %d %s\n",
			resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)

		for key, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(os.Stderr, "< %s: %s\n", key, value)
			}
		}
		fmt.Fprintf(os.Stderr, "< \n")
	}

	// Only read body if not HEAD request and not in silent mode
	if !(*head || *headLong) && !*silent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
			os.Exit(1)
		}
		// Print response body to stdout
		fmt.Print(string(body))
	}

	// Exit code based on HTTP status
	if resp.StatusCode >= 400 {
		os.Exit(1)
	}
}

// Basic URL validation
func isValidURL(url string) bool {
	return len(url) > 7 && (url[:7] == "http://" || url[:8] == "https://")
}

// Show headers only (HEAD request)
// go run main.go -I https://httpbin.org/get
// go run main.go --head https://google.com
//
// Combine with verbose for more details
// go run main.go -I -v https://httpbin.org/get
//
// Regular request with headers and body
// go run main.go -v https://httpbin.org/get
//
// Silent HEAD request (useful for scripting)
// go run main.go -I -s https://httpbin.org/get && echo "Server is up!"
