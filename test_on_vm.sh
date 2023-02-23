make build
scp ./telegraf tcgw-lb:~
ssh tcgw-lb '
    sudo bash /home/azureuser/setup_custom_telegraf.sh
'