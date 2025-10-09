# Netpala (Impala Go Edition)

A lightweight (hopefully), terminal-friendly **NetworkManager + wpa_supplicant** wrapper written in **Go**.
It’s a clone of **Impala** because Impala's UI makes me drool.

---

## 🚀 Features (so far)

- ✅ Lists available **network devices**
- ✅ Displays **known** and **scanned** networks
- ⚙️ Uses **DBus** to talk directly to NetworkManager and wpa_supplicant

---

## ⚠️ What’s Missing / TODO

- ⏳ Adding new networks
- 🔄 Forcing a scan for nearby networks
- 🐛 Probably some bugs
- 🧹 Cleanup for when I care more than I currently do (don’t count on it)

It’s functional enough for me right now, but PRs are welcome if you want to polish it up.

---

## 🧩 Implementation Notes

The DBus code (and this readme idc, sue me) was **vibe-coded**.
Yes, really. It works, I don’t care, and it’s not that deep.
If that sets off your peter tingle, feel free to fork it, rewrite it, or whisper sweet refactors to it in your own repo.

---

## 🛠️ Build & Run

\# Clone and build

\```
git clone https://github.com/joel-sgc/netpala.git
cd netpala
go build
./netpala
\```

You’ll need:

- Go 1.25.1+ (New to go, but this is the version I used so hopefully it works for you too)
- NetworkManager running
- dbus available

---

## 🧾 License

**Do What the Fuck You Want To Public License (WTFPL License)**

---

## ❤️ Closing Thoughts

I built this for myself because I wanted something that just works — and it does.
If you like it, awesome. If not, feel free to improve it or ignore it entirely.
