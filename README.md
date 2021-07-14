# TLS Secret Injector

Listen for Ingresses object created and patch them to have a valid certificate.


## Installation

```
$ git clone https://github.com/Werkspot/tls-secret-injector
$ helm upgrade tls-secret-injector --namespace tls-secret-injector --values helm/values.yaml tls-secret-injector/helm
```
