echo copy files
rm /usr/local/bin/rtty
cp rtty /usr/local/bin/
cp rtty.service /etc/systemd/system/
chmod 0644 /etc/systemd/system/rtty.service

echo install systemd
systemctl daemon-reload
systemctl enable rtty.service
systemctl retart rtty.service
