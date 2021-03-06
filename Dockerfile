FROM ubuntu:18.04

RUN ln -fs /usr/share/zoneinfo/Europe/London /etc/localtime
RUN apt-get update -y && apt-get upgrade -y && apt-get install wget curl vim sudo git -y
RUN apt-get install nginx -y

# Golang
RUN apt-get install software-properties-common -y && \
    add-apt-repository ppa:longsleep/golang-backports -y && \
    apt-get update -y && \
    apt-get install golang-go -y

# # Create the "fancast" user
# # Give build access to this env, passed in via docker build
ARG AUTH_KEY
RUN useradd -c "Fancast account" -d /home/fancast -s /bin/bash fancast
# # RUN mkdir -p /home/fancast/.ssh/ && touch /home/fancast/.ssh/authorized_keys && echo $AUTH_KEY > /home/fancast/.ssh/authorized_keys

# Node
RUN curl -sL https://deb.nodesource.com/setup_10.x | sudo -E bash -
RUN apt-get install -y nodejs

# Yarn
RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
RUN apt-get update && apt-get install yarn

# Make fancast a sudoer
RUN echo 'fancast ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

# USER fancast

RUN export PATH=/usr/lib/go-1.11/bin:$PATH
RUN echo "export PATH=/usr/lib/go-1.11/bin:$PATH" >> ~/.bashrc

# Postgresql
RUN wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ bionic-pgdg main" > /etc/apt/sources.list.d/pgdg.list
RUN sudo apt-get update -y
RUN sudo apt-get install postgresql-11 -y
RUN echo "listen_addresses = '*'" >> /etc/postgresql/11/main/postgresql.conf
RUN echo "host    all             all              0.0.0.0/0              md5" >> /etc/postgresql/11/main/pg_hba.conf
RUN echo "host    all             all              ::/0                   md5" >> /etc/postgresql/11/main/pg_hba.conf


ENV GOPATH /usr/local
ENV GO111MODULE=on
# Used by Viper Config
ENV APP_DIR /usr/local/src/bitbucket.org/jayflux/mypodcasts_injest

COPY . /usr/local/src/bitbucket.org/jayflux/mypodcasts_injest
# RUN service postgresql start && \
# sudo -u postgres psql -f /home/fancast/src/bitbucket.org/jayflux/mypodcasts_injest/build/create_database.sql

# Change to the fancast user and its home folder and run the entry point script
WORKDIR /usr/local/src/bitbucket.org/jayflux/mypodcasts_injest

# Give build access to these envs, passed in via docker build
# Do Not Remove - without these the database cannot update itself during build
ARG SPACES_KEY
ARG SPACES_SECRET_KEY

RUN /usr/lib/go-1.11/bin/go get -u ./... && \
/usr/lib/go-1.11/bin/go mod vendor && \
/usr/lib/go-1.11/bin/go build

RUN service postgresql start && sudo -u postgres psql -c "CREATE USER fancast WITH PASSWORD 'dev';" && sudo -u postgres psql -c "ALTER USER fancast WITH SUPERUSER;" && sudo -u postgres psql -c "CREATE DATABASE fancast OWNER fancast;"
RUN mkdir /var/log/fancast
RUN service postgresql start && ./mypodcasts_injest -db update


ENTRYPOINT /usr/local/src/bitbucket.org/jayflux/mypodcasts_injest/build/entrypoint_ci

EXPOSE 8060
EXPOSE 5432
