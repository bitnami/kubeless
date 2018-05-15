local k = import "ksonnet.beta.1/k.libsonnet";

local objectMeta = k.core.v1.objectMeta;
local deployment = k.apps.v1beta1.deployment;
local container = k.core.v1.container;
local service = k.core.v1.service;
local serviceAccount = k.core.v1.serviceAccount;
local configMap = k.core.v1.configMap;

local namespace = "kubeless";
local controller_account_name = "controller-acct";

local controllerEnv = [
  {
    name: "KUBELESS_INGRESS_ENABLED",
    valueFrom: {configMapKeyRef: {"name": "kubeless-config", key: "ingress-enabled"}}
  },
  {
    name: "KUBELESS_SERVICE_TYPE",
    valueFrom: {configMapKeyRef: {"name": "kubeless-config", key: "service-type"}}
    },
  {
     name: "KUBELESS_NAMESPACE",
     valueFrom: {fieldRef: {fieldPath: "metadata.namespace"}}
   },
   {
     name: "KUBELESS_CONFIG",
     value: "kubeless-config"
   },
];

local controllerContainer =
  container.default("kubeless-controller-manager", "bitnami/kubeless-controller-manager:latest") +
  container.imagePullPolicy("IfNotPresent") +
  container.env(controllerEnv);

local kubelessLabel = {kubeless: "controller"};

local controllerAccount =
  serviceAccount.default(controller_account_name, namespace);

local controllerDeployment =
  deployment.default("kubeless-controller-manager", controllerContainer, namespace) +
  {metadata+:{labels: kubelessLabel}} +
  {spec+: {selector: {matchLabels: kubelessLabel}}} +
  {spec+: {template+: {spec+: {serviceAccountName: controllerAccount.metadata.name}}}} +
  {spec+: {template+: {metadata: {labels: kubelessLabel}}}};

local crd = [
  {
    apiVersion: "apiextensions.k8s.io/v1beta1",
    kind: "CustomResourceDefinition",
    metadata: objectMeta.name("functions.kubeless.io"),
    spec: {group: "kubeless.io", version: "v1beta1", scope: "Namespaced", names: {plural: "functions", singular: "function", kind: "Function"}},
    description: "Kubernetes Native Serverless Framework",
  },
  {
    apiVersion: "apiextensions.k8s.io/v1beta1",
    kind: "CustomResourceDefinition",
    metadata: objectMeta.name("httptriggers.kubeless.io"),
    spec: {group: "kubeless.io", version: "v1beta1", scope: "Namespaced", names: {plural: "httptriggers", singular: "httptrigger", kind: "HTTPTrigger"}},
    description: "CRD object for HTTP trigger type",
  },
  {
    apiVersion: "apiextensions.k8s.io/v1beta1",
    kind: "CustomResourceDefinition",
    metadata: objectMeta.name("cronjobtriggers.kubeless.io"),
    spec: {group: "kubeless.io", version: "v1beta1", scope: "Namespaced", names: {plural: "cronjobtriggers", singular: "cronjobtrigger", kind: "CronJobTrigger"}},
    description: "CRD object for HTTP trigger type",
  }
];

local deploymentConfig = '{}';

local runtime_images ='[
  {
    "ID": "python",
    "compiled": false,
    "versions": [
      {
        "name": "python27",
        "version": "2.7",
        "runtimeImage": "kubeless/python@sha256:07cfb0f3d8b6db045dc317d35d15634d7be5e436944c276bf37b1c630b03add8",
        "initImage": "python:2.7"
      },
      {
        "name": "python34",
        "version": "3.4",
        "runtimeImage": "kubeless/python@sha256:f19640c547a3f91dbbfb18c15b5e624029b4065c1baf2892144e07c36f0a7c8f",
        "initImage": "python:3.4"
      },
      {
        "name": "python36",
        "version": "3.6",
        "runtimeImage": "kubeless/python@sha256:0c9f8f727d42625a4e25230cfe612df7488b65f283e7972f84108d87e7443d72",
        "initImage": "python:3.6"
      }
    ],
    "depName": "requirements.txt",
    "fileNameSuffix": ".py"
  },
  {
    "ID": "nodejs",
    "compiled": false,
    "versions": [
      {
        "name": "node6",
        "version": "6",
        "runtimeImage": "kubeless/nodejs@sha256:0a8a72af4cc3bfbfd4fe9bd309cbf486e7493d0dc32a691673b3f0d3fae07487",
        "initImage": "node:6.10"
      },
      {
        "name": "node8",
        "version": "8",
        "runtimeImage": "kubeless/nodejs@sha256:76ee28dc7e3613845fface2d1c56afc2e6e2c6d6392c724795a7ccc2f5e60582",
        "initImage": "node:8"
      }
    ],
    "depName": "package.json",
    "fileNameSuffix": ".js"
  },
  {
    "ID": "ruby",
    "compiled": false,
    "versions": [
      {
        "name": "ruby24",
        "version": "2.4",
        "runtimeImage": "kubeless/ruby@sha256:0dce29c0eb2a246f7d825b6644eeae7957b26f2bfad2b7987f2134cc7b350f2f",
        "initImage": "bitnami/ruby:2.4"
      }
    ],
    "depName": "Gemfile",
    "fileNameSuffix": ".rb"
  },
  {
    "ID": "php",
    "compiled": false,
    "versions": [
      {
        "name": "php72",
        "version": "7.2",
        "runtimeImage": "kubeless/php@sha256:b605bb6b5ae3b1a2a93570939296618904259d7767a14002fa9733e66d59849b",
        "initImage": "composer:1.6"
      }
    ],
    "depName": "composer.json",
    "fileNameSuffix": ".php"
  },
  {
    "ID": "go",
    "compiled": true,
    "versions": [
      {
        "name": "go1.10",
        "version": "1.10",
        "runtimeImage": "kubeless/go@sha256:e2fd49f09b6ff8c9bac6f1592b3119ea74237c47e2955a003983e08524cb3ae5",
        "initImage": "kubeless/go-init@sha256:d0812c4e8351bfd95d0574efd23613cff2664d6a57af4ed0a20ebc651382d476"
      }
    ],
    "depName": "Gopkg.toml",
    "fileNameSuffix": ".go"
  },
  {
    "ID": "csharpx",
    "compiled": false,
    "versions": [
      {
        "name": "csharpx",
        "version": "2.8.0",
        "runtimeImage": "from/kubeless-csharpx@sha256:77155dde67a81544763b7198100e085e83d04fe3ad8fcee8f9756914141a52e2",
        "initImage": "microsoft/aspnetcore-build:2.0",
      }
    ],
    "depName": "packages.csproj",
    "fileNameSuffix": ".csx"
  }
]';

local kubelessConfig  = configMap.default("kubeless-config", namespace) +
    configMap.data({"ingress-enabled": "false"}) +
    configMap.data({"service-type": "ClusterIP"})+
    configMap.data({"deployment": std.toString(deploymentConfig)})+
    configMap.data({"runtime-images": std.toString(runtime_images)})+
    configMap.data({"enable-build-step": "false"})+
    configMap.data({"function-registry-tls-verify": "true"})+
    configMap.data({"provision-image": "kubeless/unzip@sha256:f162c062973cca05459834de6ed14c039d45df8cdb76097f50b028a1621b3697"})+
    configMap.data({"provision-image-secret": ""})+
    configMap.data({"builder-image": "kubeless/function-image-builder:latest"})+
    configMap.data({"builder-image-secret": ""});

{
  controllerAccount: k.util.prune(controllerAccount),
  controller: k.util.prune(controllerDeployment),
  crd: k.util.prune(crd),
  cfg: k.util.prune(kubelessConfig),
}
