FROM golang:1.10.3-alpine3.8 as builder
MAINTAINER Alex <alex.v.chen@gmail.com>

RUN apk add --no-cache --update git bash make
ARG REPO=$GOPATH/src/github.com/hyperledger/burrow
COPY . $REPO
WORKDIR $REPO

RUN make build


FROM golang:1.10.3-alpine3.8
RUN apk add --no-cache --update git bash make

# solidity
ARG SOLC_URL=https://github.com/ethereum/solidity/releases/download/v0.4.24/solc-static-linux
ARG SOLC_BIN=/usr/bin/solc
RUN wget -O $SOLC_BIN $SOLC_URL && chmod +x $SOLC_BIN

# bos
ARG BOS_REPO=/go/src/github.com/monax/bosmarmot
RUN go get -d github.com/monax/bosmarmot || true
RUN cd $BOS_REPO && make build && make install

ARG REPO=/go/src/github.com/hyperledger/burrow
#ENV USER monax
#ENV MONAX_PATH /home/$USER/.monax
#RUN addgroup -g 101 -S $USER && adduser -S -D -u 1000 $USER $USER
#WORKDIR $MONAX_PATH
#USER $USER:$USER
COPY --from=builder $REPO/bin/* /usr/local/bin/

EXPOSE 26656
EXPOSE 26658
EXPOSE 10997

CMD [ "burrow" ]