FROM golang:latest
RUN go install github.com/go-delve/delve/cmd/dlv@latest

FROM amd64/ubuntu:22.04
ARG DEBIAN_FRONTEND=noninteractive
ENV TZ=UTC
RUN apt-get update && \
    apt-get install -y wget curl openjdk-17-jre && \
    wget https://s3.us-west-2.amazonaws.com/downloads.yugabyte.com/repos/reporpms/yb-apt-repo_1.0.0_all.deb && \
    apt-get install -y ./yb-apt-repo_1.0.0_all.deb && \
    apt-get update && \
    apt-get upgrade -y binutils && \
    curl -sL https://aka.ms/InstallAzureCLIDeb | bash && \
    mkdir -p /opt/yb-voyager && \
    apt-get install -y yb-voyager && \
    rm -rf /yb-apt-repo_1.0.0_all.deb && \
    apt-get clean
COPY --from=0 $HOME/go/bin/dlv /usr/local/bin
CMD [“yb-voyager”]
