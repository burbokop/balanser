
#!/bin/sh

echo "Running build ..."
pwd

go get -u github.com/burbokop/design-practice-1/build/cmd/bood

echo "ls $GOPATH/bin ..."
ls $GOPATH/bin

export PATH=$PATH:$GOPATH/bin

bood