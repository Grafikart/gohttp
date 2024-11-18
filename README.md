# Comprendre les différentes version du protocole HTTP

Afin de mieux comprendre les différences qu'il existe entre les mutliples versions du protocole HTTP j'ai décidé d'essayer d'implémenter un serveur HTTP 1, 2 et 3 en utilisant golang.

```
go run . http1
go run . http1
go run . http3
```

Ce code n'a pas vocation a être utilisé en tant que tel mais a une vocation pédagogique.

## Source d'informations

- [RFC HTTP/1](https://datatracker.ietf.org/doc/html/rfc2616)
- [RFC HTTP/2](https://datatracker.ietf.org/doc/html/rfc9113)
- [RFC HTTP/3](https://datatracker.ietf.org/doc/html/rfc9114)
- [RFC Quic](https://datatracker.ietf.org/doc/html/rfc9000)
- [QUIC breakdown](https://quic.xargs.org/)
- [HTTP/3 explained](https://http3-explained.haxx.se/en)
- [Explication TCP / UDP](https://www.gipsa-lab.grenoble-inp.fr/~christian.bulfone/MIASHS-L3/PDF/3-Les_protocoles_UDP_TCP.pdf)