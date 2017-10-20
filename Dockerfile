FROM registry.centos.org/che-stacks/centos-stack-base

LABEL maintainer "Kamesh Sampath<kamesh.sampath@hotmail.com>"

COPY ./bin/checontroller-linux-amd64 /checontroller

ENTRYPOINT [ "/checontroller","--incluster" ]
