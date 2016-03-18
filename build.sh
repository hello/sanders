GIT_COMMIT=$(git rev-parse HEAD)


echo "--> Building..."
go build -ldflags "${CGO_LDFLAGS} -X main.GitCommit=${GIT_COMMIT}" -v
cp sanders /usr/local/bin/
