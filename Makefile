default:
	GOOS=darwin GOARCH=amd64 go build -a -v -x 
	sudo install -m0755 SaferViewer /Applications/SaferViewer.app/Contents/MacOS 

clean:
	go clean -a -v -x
