FROM scratch

LABEL author "Kamesh Sampath<kamesh.sampath@hotmail.com>"

COPY ./bin/checontroller-linux-amd64 /checontroller
ENTRYPOINT [ "checontroller","-incluster" ]
