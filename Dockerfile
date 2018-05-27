from ubuntu:17.10

RUN apt-get update && apt-get upgrade -y && apt-get install wget curl vim sudo git -y
RUN apt-get install nginx -y

# Golang
RUN apt-get install software-properties-common -y && \
add-apt-repository ppa:gophers/archive -y && \
apt-get update -y && \
apt-get install golang-1.10-go -y
RUN export PATH=/usr/lib/go-1.10/bin:$PATH
RUN echo "export PATH=/usr/lib/go-1.10/bin:$PATH" >> ~/.bashrc

# Postgresql
RUN echo "deb http://apt.postgresql.org/pub/repos/apt/ zesty-pgdg main" > /etc/apt/sources.list.d/pgdg.list
RUN wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN apt-get update -y
RUN apt-get install postgresql-10 -y

# Create the "fancast" user
# Give build access to this env, passed in via docker build
ARG AUTH_KEY
RUN useradd -c "Fancast account" -d /home/fancast fancast
RUN mkdir -p /home/fancast/.ssh/ && touch /home/fancast/.ssh/authorized_keys && echo $AUTH_KEY > /home/fancast/.ssh/authorized_keys

# Make fancast a sudoer
RUN echo 'fancast ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

ENV GOPATH /home/fancast

COPY . /home/fancast/src/bitbucket.org/jayflux/mypodcasts_injest
# RUN service postgresql start && \
# sudo -u postgres psql -f /home/fancast/src/bitbucket.org/jayflux/mypodcasts_injest/build/create_database.sql

# Change to the fancast user and its home folder and run the entry point script
WORKDIR /home/fancast/src/bitbucket.org/jayflux/mypodcasts_injest

# Give build access to these envs, passed in via docker build
ARG SPACES_KEY
ARG SPACES_SECRET_KEY

ENV PATH /home/fancast/bin:$PATH
RUN /usr/lib/go-1.10/bin/go get -u github.com/golang/dep/cmd/dep
RUN /home/fancast/bin/dep ensure
RUN /usr/lib/go-1.10/bin/go build
RUN sudo service postgresql start && sudo -u postgres psql -c "CREATE USER fancast WITH PASSWORD 'dev';" && sudo -u postgres psql -c "ALTER USER fancast WITH SUPERUSER;" && sudo -u postgres psql -c "CREATE DATABASE fancast OWNER fancast;"
RUN service postgresql start && chown -R fancast:fancast /home/fancast && sudo -E -u fancast ./mypodcasts_injest -db update
RUN chown -R fancast:fancast /home/fancast

USER fancast

ENTRYPOINT /home/fancast/src/bitbucket.org/jayflux/mypodcasts_injest/build/entrypoint_ci

EXPOSE 80
EXPOSE 5432
