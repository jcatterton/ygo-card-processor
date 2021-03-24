#!/bin/bash
VERSION="$(cat version)"
CHART_REF="ygo-card-processor"

CONTEXT="$(kubectl config current-context)"
if [[ ${CONTEXT} != "justin" ]]; then
echo "Incorrect kubernetes context"
exit 1
fi

echo "Updating to version ${VERSION}..."
sed -i '' -E s/tag:\ [0-9]*\.[0-9]*\.[0-9]*\([-][A-Za-z0-9.-]+\)?/tag:\ ${VERSION}/g ./${CHART_REF}/values.yaml
sed -i '' -E s/version:\ [0-9]*\.[0-9]*\.[0-9]*\([-][A-Za-z0-9.-]+\)?/version:\ ${VERSION}/g ./${CHART_REF}/Chart.yaml

build() {
    docker build -f ./docker/Dockerfile -t "192.168.1.15:5000/ygo-card-processor:${VERSION}" .
}

push() {
    docker push "192.168.1.15:5000/ygo-card-processor:${VERSION}"
}

deploy() {
    helm upgrade -i ${CHART_REF} ./${CHART_REF}
}

if ! build; then
echo "Error building image"
exit 1
fi

if ! push; then
echo "Error pushing image"
exit 1
fi

if ! deploy; then
echo "Error deploying helm release"
exit 1
fi
