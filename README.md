# mtest

> A code that cannot be tested is flawed – Anonymous

Mtest will cater to various testing needs of openebs projects. Mtest can be 
easily guided to try out a particular customer use case. On a simpler scheme 
of things Mtest will be used during CI runs.

## Moving parts of mtest

- Will include a CLI
- Will read a config file (WIP)
- Concept of Runners, Drivers & Executors (WIP)


## Design References/Credits

> Mtest has its design elements influenced from following libraries:

- hashicorp/nomad
- aws/aws-sdk-go
- rancher/convoy
- k8s e2e concepts
