
#!/bin/sh

echo "Running build ..."
pwd

go get github.com/burbokop/design-practice-1

echo "ls $GOPATH/bin ..."
ls $GOPATH/bin

export PATH=$PATH:$GOPATH/bin

bood