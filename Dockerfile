FROM scratch

LABEL author "Kamesh Sampath<kamesh.sampath@hotmail.com>"

COPY ./bin/che-stack-refresher-linux-amd64 /che-stack-refresher

ENTRYPOINT [ "che-stack-refresher" ]
