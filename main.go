package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var upload_root_git = "/home/targit"
var delete = true
var Version = ""

func backup(w http.ResponseWriter, r *http.Request) {
	//var upload_root string

	system_arr, system_found := r.Header["System"]
	fmt.Println("system", system_arr)
	//if system_found {
	//	upload_root = upload_root + "/" + system[0]
	//} else
	if !system_found || len(system_arr) == 0 || len(system_arr[0]) == 0 {
		// If no certificate is found, print error and return
		fmt.Fprintf(w, "Please make sure you specify the system name when POSTING.")
		return
	}
	system := system_arr[0]

	switch r.Method {
	case "GET":
		// When presented with a GET request, just send some feedback that is easy to understand
		// take for example someone opens this in a browser.
		fmt.Fprintf(w, "TarGit acquired.")

	case "POST":
		upload_root_han, err := ioutil.TempFile("/dev/shm", "system-backup")
		if err != nil {
			log.Fatal(err)
		}
		upload_root := upload_root_han.Name()
		//working_dir, _ := os.Getwd()
		os.Remove(upload_root_han.Name())
		os.Mkdir(upload_root, 0755)
		os.Chdir(upload_root)
		defer RemoveContents(upload_root)

		// We'll parse the posted file by first passing the Body to a gzip reader:
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			fmt.Fprintf(w, "Please make sure you sent an archive; error gunzip'ing the input stream: %v", err)
			return
		}
		// Then pass the unzipped version to a tar reader:
		tarReader := tar.NewReader(gzipReader)

		// Checkout the branch or create the branch
		out, err := exec.Command("/usr/bin/git", "--work-tree="+upload_root, "--git-dir="+upload_root_git, "checkout", system).Output()
		if err != nil {
			log.Println("Branch doesn't exists, trying to create one", string(out))

			log.Println("/usr/bin/git", "--work-tree="+upload_root, "--git-dir="+upload_root_git, "checkout", "--orphan", system)
			out, err = exec.Command("/usr/bin/git", "--work-tree="+upload_root, "--git-dir="+upload_root_git, "checkout", "--orphan", system).Output()
			if err != nil {
				log.Println("Error running git checkout to create new branch:", string(out), err)
				return
			}
		}

		// Extract the files from the tar one by one and put them into a subfolder:
		for i := 0; ; i++ {
			header, err := tarReader.Next()

			if err == io.EOF {
				log.Println("Ran to end of file on TAR", err)
				break // if we are at the end of the tar file break the for loop
			}
			if err != nil {
				log.Println("An error occured when extracting:", err)
				break
			}

			name := header.Name
			_, fname := filepath.Split(header.Name)
			// avoid malformed and dotted names
			if len(fname) > 0 && fname[0] == '.' {
				continue
			}

			if debug {
				log.Println("found type", header.Typeflag)
			}

			switch header.Typeflag {
			case tar.TypeDir: // = directory
				if debug {
					log.Println("Creating Directory:", name)
				}
				os.Mkdir(upload_root+"/"+name, 0755)
				os.Chtimes(upload_root+"/"+name, header.AccessTime, header.ModTime)
			case tar.TypeReg: // = regular file
				if debug {
					log.Println("Extracting file:", name)
				}
				data := make([]byte, header.Size)
				_, err := tarReader.Read(data)
				if err != nil && err != io.EOF {
					log.Println("An error occured when reading in file:", name, err)
					break
				}

				ioutil.WriteFile(upload_root+"/"+name, data, 0644)
				os.Chtimes(upload_root+"/"+name, header.AccessTime, header.ModTime)
			default:
				log.Printf("%s : %c %s %s\n",
					"Yikes! Unable to figure out type",
					header.Typeflag,
					"in file",
					name,
				)
			}
		}

		if debug {
			log.Println("git", "--work-tree="+upload_root, "--git-dir="+upload_root_git, "add", "--all")
		}

		_, err = exec.Command("git", "--work-tree="+upload_root, "--git-dir="+upload_root_git, "add", "--all").Output()
		if err != nil {
			log.Println("Error running git add command:", err)
			return
		}

		_, err = exec.Command("git", "--work-tree="+upload_root, "--git-dir="+upload_root_git, "commit", "-m", "Automatic backup "+(time.Now().String())).Output()
		if err != nil {
			log.Println("Git command failed to commit:", err)
			return
		}
	/*
		out, err := exec.Command("git", "--git-dir="+upload_root, "push").Output()
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

var debug bool

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Push TarGIT.  (%s), written by Paul Schou github@paulschou.com December 2020\nPrsonal use only, provided AS-IS -- not responsible for loss.\nUsage implies agreement.\n\n Usage of %s:\n", Version, os.Args[0])
		flag.PrintDefaults()
	}
	var listen = flag.String("listen", ":8080", "Listen address tgz push")
	var prefix = flag.String("prefix", "", "URL prefix used upstream in reverse proxy endpoint for all incoming requests")
	var verbose = flag.Bool("debug", false, "Verbose output")
	var upload_dir = flag.String("upload_dir", upload_root_git, "Path for the git repository")
	var delete_files = flag.Bool("delete_tmp_dir", delete, "Delete temporary upload directory after upload")
	flag.Parse()

	delete = *delete_files
	upload_root_git = *upload_dir
	//var err error
	debug = *verbose

	urlPrefix := "/" + strings.TrimRight(*prefix, "/")

	http.HandleFunc(urlPrefix, backup)

	// Now that all the files have been extracted to upload_root, use git to switch
	// to the branch of the config and push changes
	out, err := exec.Command("/usr/bin/git", "version").Output()
	if err != nil {
		log.Println("Error running git version command:", err)
		return
	}
	log.Println("Git version", string(out))

	// Make sure the repo is init
	out, err = exec.Command("/usr/bin/git", "--git-dir="+upload_root_git, "init").Output()
	if err != nil {
		log.Println("Error running git init command:", err)
		return
	}
	log.Println(string(out))

	fmt.Println("To print out the list of current system branchs use:\n/usr/bin/git", "--git-dir="+upload_root_git, "branch\n")
	fmt.Println("To print out the list of current files in a branch use:\n/usr/bin/git", "--git-dir="+upload_root_git, "ls-tree --full-name -r system_name\n")

	fmt.Printf("Starting server for HTTP POST (on %s%s) of a tar to send to git (aka: tar'git)...\n", *listen, urlPrefix)
	if err := http.ListenAndServe(*listen, nil); err != nil {
		log.Fatal(err)
	}
}

func RemoveContents(dir string) error {
	defer os.Chdir("/tmp")
	fmt.Println("removing directory", dir)
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		fmt.Println(filepath.Join(dir, name))
		if debug {
			log.Println("removing", filepath.Join(dir, name))
		}
		//err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			fmt.Println("err:", err)
			return err
		}
	}
	err = os.RemoveAll(dir)
	return nil
}
