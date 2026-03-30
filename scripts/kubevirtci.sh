#!/bin/bash

set -ex

export KUBEVIRT_MEMORY_SIZE="${KUBEVIRT_MEMORY_SIZE:-16G}"
export KUBEVIRT_REPO="${KUBEVIRT_REPO:-https://github.com/kubevirt/kubevirt-aie.git}"
export KUBEVIRT_BRANCH="${KUBEVIRT_BRANCH:-release-1.8-aie-nv}"
export KUBEVIRT_CENTOS_STREAM_VERSION="${KUBEVIRT_CENTOS_STREAM_VERSION:-10}"
export KUBEVIRT_CS10_BUILDER_VERSION="${KUBEVIRT_CS10_BUILDER_VERSION:-2602251001-25ce1ccb15}"
export NAMESPACE="${NAMESPACE:-kubevirt}"

_base_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
_kubevirt_dir="${_base_dir}/_kubevirt"
_cluster_up_dir="${_kubevirt_dir}/kubevirtci/cluster-up"
_kubectl="${_cluster_up_dir}/kubectl.sh"
_cli="${_cluster_up_dir}/cli.sh"
_action=$1
shift

function determine_cri_bin() {
    "${_base_dir}/hack/container-engine.sh"
}

function kubevirtci::install() {
    if [[ ! -d "${_kubevirt_dir}" ]]; then
        git clone --depth 1 --branch "${KUBEVIRT_BRANCH}" "${KUBEVIRT_REPO}" "${_kubevirt_dir}"
    fi
}

function kubevirtci::up() {
    make cluster-up -C "${_kubevirt_dir}"
    make cluster-sync -C "${_kubevirt_dir}"

    echo "waiting for kubevirt to become ready, this can take a few minutes..."
    ${_kubectl} -n "${NAMESPACE}" wait kv kubevirt --for condition=Available --timeout=15m
}

function kubevirtci::down() {
    make cluster-down -C "${_kubevirt_dir}"
}

function kubevirtci::sync() {
    local cri
    cri=$(determine_cri_bin)
    if [[ -z "${cri}" ]]; then
        echo >&2 "no working container runtime found. Neither docker nor podman seems to work."
        exit 1
    fi

    local docker_tag="${DOCKER_TAG:-devel}"
    local docker_prefix="${DOCKER_PREFIX:-quay.io/kubevirt}"
    local image_name="${IMAGE_NAME:-iommufd-device-plugin}"
    local img="${docker_prefix}/${image_name}:${docker_tag}"

    echo "Building container image ${img}..."
    ${cri} build -t "${img}" "${_base_dir}"

    echo "Loading image into cluster..."
    local registry_port
    registry_port=$(${_cli} ports registry 2>/dev/null || true)
    if [[ -n "${registry_port}" ]]; then
        local registry="localhost:${registry_port}"
        local registry_img="${registry}/${image_name}:${docker_tag}"
        ${cri} tag "${img}" "${registry_img}"
        ${cri} push "${registry_img}" --tls-verify=false 2>/dev/null || \
            ${cri} push "${registry_img}"
        img="registry:5000/${image_name}:${docker_tag}"
    fi

    local kubeconfig
    kubeconfig=$(kubevirtci::kubeconfig)

    echo "Deploying iommufd-device-plugin DaemonSet..."
    sed "s|quay.io/kubevirt/iommufd-device-plugin:latest|${img}|" \
        "${_base_dir}/deploy/daemonset.yaml" | \
        KUBECONFIG="${kubeconfig}" ${_kubectl} apply -f -
    KUBECONFIG="${kubeconfig}" ${_kubectl} rollout status daemonset/iommufd-device-plugin \
        -n kube-system --timeout=2m

    echo "iommufd-device-plugin synced to cluster."
}

function kubevirtci::kubeconfig() {
    "${_cluster_up_dir}/kubeconfig.sh"
}

function kubevirtci::kubectl() {
    ${_kubectl} "$@"
}

kubevirtci::install

case ${_action} in
    "up")
        kubevirtci::up
        ;;
    "down")
        kubevirtci::down
        ;;
    "sync")
        kubevirtci::sync
        ;;
    "kubeconfig")
        kubevirtci::kubeconfig
        ;;
    "kubectl")
        kubevirtci::kubectl "$@"
        ;;
    *)
        echo "Unknown command '${_action}'. Known commands: up, down, sync, kubeconfig, kubectl"
        exit 1
        ;;
esac
