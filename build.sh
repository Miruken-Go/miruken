docker run -v $(pwd):/go/src --workdir=/go/src golang:1.20 go test ./...

TAG=$(date +%s); echo $TAG
git tag $TAG
git tag
git push origin $TAG