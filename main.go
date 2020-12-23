package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var upload_root = "/dev/shm"

func backup(w http.ResponseWriter, r *http.Request) {
	var host_folder string

	// We'll use "User-Agent" header for now, but this shall be replaced with "X-User" from mTLS certificate
	// once the request is passed through the reverse proxy...
	user, user_found := r.Header["User-Agent"]
	if user_found {
		host_folder = upload_root + "/" + user[0][0:5]
		os.Mkdir(host_folder, 0755) // make sure the user folder is created
	} else if !user_found || user[0] == "" {
		// If no certificate is found, print error and return
		fmt.Fprintf(w, "Please make sure your mTLS certificate is being sent to identify who you are")
		return
	}

	switch r.Method {
	case "GET":
		// When presented with a GET request, just send some feedback that is easy to understand
		// take for example someone opens this in a browser.
		fmt.Fprintf(w, "The header says you are %v.", user)

	case "POST":
		// We'll parse the posted file by first passing the Body to a gzip reader:
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			fmt.Fprintf(w, "Please make sure you sent an archive; error gunzip'ing the input stream: %v", err)
			return
		}
		// Then pass the unzipped version to a tar reader:
		tarReader := tar.NewReader(gzipReader)

		// Extract the files from the tar one by one and put them into a subfolder:
		for i := 0; ; i++ {
			header, err := tarReader.Next()

			if err == io.EOF {
				break // if we are at the end of the tar file break the for loop
			}
			if err != nil {
				log.Println("An error occured when extracting:", err)
				break
			}

			name := header.Name
			switch header.Typeflag {
			case tar.TypeDir: // = directory
				log.Println("Creating Directory:", name)
				os.Mkdir(host_folder+name, 0755)
			case tar.TypeReg: // = regular file
				log.Println("Extracting file:", name)
				data := make([]byte, header.Size)
				_, err := tarReader.Read(data)
				if err != nil && err != io.EOF {
					log.Println("An error occured when reading in file:", name, err)
					break
				}

				ioutil.WriteFile(host_folder+name, data, 0755)
			default:
				log.Printf("%s : %c %s %s\n",
					"Yikes! Unable to figure out type",
					header.Typeflag,
					"in file",
					name,
				)
			}
		}

		// Now that all the files have been extracted to host_folder, use git to switch
		// to the branch of the config and push changes
		out, err := exec.Command("/usr/bin/git", "version").Output()
		if err != nil {
			log.Println("Error running git version command:", err)
			return
		}
		log.Println("Git version", string(out))
		/*
			_, err = exec.Command("git", "--git-dir="+host_folder, "add", "--all").Output()
			if err != nil {
				log.Println("Error running git add command:", err)
				return
			}
			_, err = exec.Command("git", "--git-dir="+host_folder, "commit", "-m", "Automatic backup").Output()
			if err != nil {
				log.Println("Error running git command to commit:", err)
				return
			}
			out, err := exec.Command("git", "--git-dir="+host_folder, "push").Output()
			if err != nil {
				log.Println("Error running git command to push:", err)
				return
			}
			log.Println("Git push successful", out)
		*/
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func main() {
	http.HandleFunc("/", backup)

	fmt.Printf("Starting server for HTTP POST of a tar to send to git (aka: tar'git)...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
