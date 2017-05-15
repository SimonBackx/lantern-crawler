# Required to run as root
# sudo su
# Port 11371 needs to be open temporary
iptables -A OUTPUT -p tcp --dport 11371 -j ACCEPT
iptables -A INPUT -p tcp --sport 11371 -j ACCEPT

iptables -A INPUT -p tcp --dport 11371 -j ACCEPT
iptables -A OUTPUT -p tcp --sport 11371 -j ACCEPT

sudo gpg --keyserver keys.gnupg.net --recv A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89
sudo gpg --export A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89 | apt-key add -

# install tor and curl
sudo apt-get update && apt-get install -qq tor deb.torproject.org-keyring

sudo service tor stop
sudo systemctl mask tor