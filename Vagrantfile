# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "grahamc-precise"
  config.vm.hostname = "route"
  config.vm.box_url = "http://grahamc.com/vagrant/ubuntu-12.04-omnibus-chef.box"
  config.vm.network :forwarded_port, guest: 6000, host: 8080

  config.vm.provision :chef_client do |chef|
    chef.chef_server_url = "https://chef.umn.edu/organizations/api"
    chef.validation_key_path = ".chef/api-validator.pem"
    chef.validation_client_name = "api-validator"
    chef.add_role "base"
    chef.add_role "route"
  end
end
