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

# Create the "developer" user
RUN useradd -c "Developer account" developer
# Make developer a sudoer
RUN echo 'developer ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

RUN mkdir /var/local/src
ENV GOPATH /var/local

COPY . /var/local/src/bitbucket.org/jayflux/mypodcasts_injest
RUN service postgresql start && \
sudo -u postgres psql -f /var/local/src/bitbucket.org/jayflux/mypodcasts_injest/build/create_database.sql && \
sudo -u developer psql mypodcasts -f /var/local/src/bitbucket.org/jayflux/mypodcasts_injest/build/create_tables.sql

# RUN cp -f /var/local/mypodcasts/docker/mypodcasts.conf /etc/nginx/sites-enabled/default

# Change to the developer user and its home folder and run the entry point script
# USER developer
WORKDIR /var/local/src/bitbucket.org/jayflux/mypodcasts_injest

ENV PATH /var/local/bin:$PATH
RUN /usr/lib/go-1.10/bin/go get -u github.com/golang/dep/cmd/dep

ENTRYPOINT /var/local/mypodcasts/docker/entrypoint_ci

EXPOSE 80
EXPOSE 5432
