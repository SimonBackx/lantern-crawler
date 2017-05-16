# Required to run as root
# sudo su
# Port 11371 needs to be open temporary
iptables -A OUTPUT -p tcp --dport 11371 -j ACCEPT
iptables -A INPUT -p tcp --sport 11371 -j ACCEPT

iptables -A INPUT -p tcp --dport 11371 -j ACCEPT
iptables -A OUTPUT -p tcp --sport 11371 -j ACCEPT

You need to add the following entry in /etc/apt/sources.list or a new file in /etc/apt/sources.list.d/:

deb http://deb.torproject.org/torproject.org xenial main
deb-src http://deb.torproject.org/torproject.org xenial main
Then add the gpg key used to sign the packages by running the following commands at your command prompt:

gpg --keyserver keys.gnupg.net --recv A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89
gpg --export A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89 | sudo apt-key add -
You can install it with the following commands:

$ apt-get update
$ apt-get install tor deb.torproject.org-keyring

sudo service tor stop
sudo systemctl mask tor