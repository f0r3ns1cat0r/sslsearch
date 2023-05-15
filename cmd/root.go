/*
Copyright © 2023 Harsh Varagiya

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

var (
	debugFlag              bool
	traceFlag              bool
	keywordRegexString     string
	regionRegexString      string
	portsString            string
	outFileName            string
	outputOverwrite        bool
	threadCount            int
	cidrSuffixPerGoRoutine int
	consoleOut             bool

	grabServerHeader           bool
	grabJarmFingerprint        bool
	serverHeaderThreadCount    int
	jarmFingerptintThreadCount int
)

var (
	log               = logrus.New()
	statsLock         = sync.RWMutex{}
	cidrRangesToScan  = 0
	cidrRangesScanned = 0
	totalIpsScanned   = 0
	totalFindings     = 0
	jarmRetryCount    = 3
	tcpTimeout        = 10

	httpClientPool = sync.Pool{
		New: func() interface{} {
			return &fasthttp.Client{
				TLSConfig: &tls.Config{
					// for server header check skip SSL validation
					InsecureSkipVerify: true,
				},
			}
		},
	}
	dialerPool = sync.Pool{
		New: func() interface{} {
			return &net.Dialer{
				Timeout: time.Duration(tcpTimeout) * time.Second,
			}
		},
	}
	tlsConfigPool = sync.Pool{
		New: func() interface{} {
			return &tls.Config{
				InsecureSkipVerify: true,
			}
		},
	}
	errConn              = fmt.Errorf("could not connect to remote host")
	errNoTls             = fmt.Errorf("could not find TLS on remote port")
	errNoMatch           = fmt.Errorf("certificate details did not match requirement")
	errCtxCancelled      = fmt.Errorf("parent context cancelled")
	errJarmNotCalculated = fmt.Errorf("error calculating JARM fingerprint")
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sslsearch",
	Short: "hunt for keywords in SSL certificates on cloud",
	Long: `search cloud providers / IP ranges to scan for 
SSL certificates. Do some initial recon for the findings if needed. 
Initial Recon: 
	1. Server Header Grabbing
	2. JARM Fingerprinting`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sslsearch.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolVarP(&debugFlag, "debug", "v", false, "enable debug logs")
	rootCmd.Flags().BoolVar(&traceFlag, "trace", false, "enable trace logs")

	// refined input flags
	rootCmd.Flags().StringVarP(&keywordRegexString, "keyword-regex", "k", ".*", "keyword regex to search in subject or SAN (ex: amazon,google). Default .* which matches all")
	rootCmd.Flags().StringVarP(&regionRegexString, "region-regex", "r", ".*", "regex of cloud service provider region to search")
	rootCmd.Flags().StringVarP(&portsString, "ports", "p", "443", "ports to search (default: 443)")
	rootCmd.Flags().StringVarP(&outFileName, "out", "o", "output.log", "output file on disk")
	rootCmd.Flags().BoolVar(&outputOverwrite, "overwrite", false, "overwrite output file if it exists")
	rootCmd.Flags().IntVarP(&threadCount, "threads", "t", 2000, "number of parallel threads to use")

	// advanced input flags
	rootCmd.Flags().IntVar(&cidrSuffixPerGoRoutine, "suffix", 4, "CIDR suffix per goroutine (each thread will scan 2^x IPs. default 4)")
	rootCmd.Flags().IntVar(&tcpTimeout, "timeout", 10, "tcp connection timeout in seconds")
	rootCmd.Flags().BoolVar(&consoleOut, "console-out", false, "actively print result JSON to console")

	// recon flags
	// server header
	rootCmd.Flags().BoolVar(&grabServerHeader, "server-header", false, "attempt enrich results by grabbing the https server header for results")
	rootCmd.Flags().IntVar(&serverHeaderThreadCount, "server-header-threads", 10, "number of threads to use for server header result enrichment")

	// JARM fingerprinting
	rootCmd.Flags().BoolVar(&grabJarmFingerprint, "jarm", false, "attempt enrich results by grabbing the JARM fingerprint")
	rootCmd.Flags().IntVar(&jarmRetryCount, "jarm-retry-count", 3, "retry attempts for JARM fingerprint (default 3)")
	rootCmd.Flags().IntVar(&jarmFingerptintThreadCount, "jarm-threads", 50, "number of threads to use for JARM fingerprint enrichment")

}
