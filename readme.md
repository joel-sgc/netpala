Requirements: 
- NetworkManager
- WPA_Supplicant
- python 3.13
- wpa_cli

---
HELLO! YOU GOTTA ADD THIS LINE TO YOUR SUDOERS FILE
```
$USER ALL=(ALL:ALL) NOPASSWD: /usr/bin/wpa_cli status
```