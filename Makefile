default:
	GOOS=darwin GOARCH=amd64 go build -a -v -x 

clean:
	go clean -a -v -x

install:
	sudo install -m0755 SaferViewer /Applications/SaferViewer.app/Contents/MacOS
	sudo install -m0644 ApplicationStub.icns /Applications/SaferViewer.app/Contents/Resources
