from fabric.api import *
from fabric.contrib.files import exists
from fabric.state import output
from fabric.contrib.project import rsync_project
from fabric.contrib.console import confirm
import yaml

output['running'] = False

env.gateway = "ubuntu@lantrn.xyz"
env.key_filename=['lantern_rsa']

env.hosts = ["ubuntu@172.26.13.178"]

def deploy():
    build()
    upload()
    print ("Done.")

def build():
    print ("Building...")
    local("mkdir -p build")
    local("env GOOS=linux GOARCH=amd64 go build -o build/crawler .")

def upload():
    print ("Uploading...")
    uploading_directory = "/etc/lantern"

    run("mkdir -p "+uploading_directory)

    run("sudo systemctl stop lanterncrawler")

    with settings(hide('warnings', 'running', 'stdout')):
        rsync_project(remote_dir=uploading_directory, local_dir="build/crawler", delete=True)

    run("sudo /etc/lantern/crawler -service uninstall")
    run("sudo /etc/lantern/crawler -service install")
    run("sudo /etc/lantern/crawler -service start")
