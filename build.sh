docker run -v $(pwd):/go/src --workdir=/go/src golang:1.20 go test ./...

TAG=$(docker run --rm -v "$(pwd):/repo" gittools/gitversion:5.12.0-alpine.3.14-6.0 /repo /showvariable SemVer)
echo "Build Version: $TAG"

git tag $TAG
git push origin $TAG