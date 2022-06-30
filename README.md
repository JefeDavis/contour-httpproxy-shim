<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# Contour HTTPProxy Support for cert-manager

This project supports automatically getting a certificate for
Contour HTTPProxies from any cert-manager Issuer.

## Prerequisites:

1) Ensure you have [cert-manager](https://github.com/cert-manager/cert-manager) installed
through the method of your choice.
For example, with the regular manifest:
```sh
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.8.0/cert-manager.yaml
```
Both **ClusterIssuer** and namespace based **Issuer** are possible. Here a **ClusterIssuer** is used:

2) For example, create the ClusterIssuer (no additional ingress class is needed for the httpproxy router. The example.com email must be replaced by another one):

```yaml
apiVersion: v1
items:
- apiVersion: cert-manager.io/v1
  kind: ClusterIssuer
  metadata:
    annotations:
    name: letsencrypt-prod
  spec:
    acme:
      email: mymail@example.com
      preferredChain: ""
      privateKeySecretRef:
        name: letsencrypt-prod
      server: https://acme-v02.api.letsencrypt.org/directory
      solvers:
      - http01:
          ingress: {}
```

```sh
kubectl apply -f clusterissuer.yaml
```

3) Make sure that there is an A record on the load balancer IP or a CNAME record on the load balancer hostname in your DNS system for the HTTP-01 subdomain. (or configured with externalDNS)

```
CNAME:
  Name: *.service.clustername.domain.com
  Alias: your-lb-domain.cloud
```

## Usage

Install in your cluster using the make file

```shell
# To run from your shell for testing
make run

# To deploy to the cluster
IMG=myregistry.com/cert-manager-contour-httpproxy:latest

make docker-build

make docker-push

make deploy
```

If you follow the above prerequisites, use this annotations below
```yaml
...
metadata:
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
...
spec:
  virtualhost:
    fqdn: app.service.clustername.domain.com
    TLS:
      secretName: app-tls
...
```


Annotate your routes:

```yaml
apiVersion: projectcontour.io/v1
kind: HTTPProxy
metadata:
  name: example-route
  annotations:
    cert-manager.io/issuer: my-issuer # This is the only required annotation to use a namespace scoped issuer
    cert-manager.io/cluster-issuer: my-issuer # This is the only required annotation to use a cluster scoped issuer
    cert-manager.io/issuer-group: cert-manager.io # Optional, defaults to cert-manager.io
    cert-manager.io/issuer-kind: Issuer # Optional, defaults to Issuer, could be ClusterIssuer or an External Issuer
spec:
  virtualHost:
    fqdn: app.service.clustername.domain.com # will be added to the Subject Alternative Names of the CertificateRequest
    TLS:
      secretName: app-tls # will be the target to store the resulting certificate in (does not have to exist yet)
    
      
```

If both of cert-manager.io/issuer and cert-manager.io/cluster-issuer exist, cluster-issuer takes precedence.


Now the website can be called: https://app.service.clustername.domain.com

# Why is This a Separate Project?

cert manager do not wish to support non Kubernetes (or kubernetes-sigs) APIs in cert-manager core. This adds
a large maintenance burden, and it's hard to e2e test everyone's CRDs. However, Contour is
widely used, so it makes sense to have some support for it in the cert-manager ecosystem.

Unfortunately, cert-manager is not really designed
to be imported as a module. It has a large number of transitive dependencies that would add an unfair
amount of maintenance to whichever project it is submitted to. In the future, cert manager has said it 
would like to split the cert-manager APIs and typed clients out of the main cert-manager repo, at which 
point it would be easier for other people to consume in their projects.

