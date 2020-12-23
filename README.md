# push_targit
Example http post endpoint with tgz extractor and shell escape

First, compile the golang code then run the binary:
```
push_targit$ make && ./post-targit
```

# Testing
Upload the file
```
push_targit$ curl --data-binary '@test.tgz' http://localhost:8080/
```
Verify that the tar-gzip'ed file has been uploaded and extracted properly:
```
push_targit$ find /dev/shm/curl/
/dev/shm/curl/
/dev/shm/curl/testfile
push_targit$ cat /dev/shm/curl/testfile
This is a test

```

An idea is to do something along the lines of a sparse checkout in the working folder
and then make the branch the actual system name that is being backed up...
```
git init <repo>
cd <repo>
git remote add origin <url>
git config core.sparsecheckout true
git pull --depth=1 origin master
```

From the server side you'd see:
```
push_targit$ ./post-targit
Starting server for HTTP POST of a tar to send to git (aka: tar'git)...
2020/12/22 21:00:46 Extracting file: testfile
2020/12/22 21:00:46 Git version git version 1.8.3.1

```
