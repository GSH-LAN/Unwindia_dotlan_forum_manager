<p align="center">
  <a href="https://github.com/gsh-lan/unwindia" target="blank"><img src="https://raw.githubusercontent.com/GSH-LAN/Unwindia/main/.resources/images/logo.png" height="128" alt="unwindia logo">
  <a href="https://github.com/gsh-lan/unwindia" target="blank"><img src="https://raw.githubusercontent.com/GSH-LAN/Unwindia/main/.resources/images/header.svg" height="128" alt="unwindia header" /></a>
</p>

# UNWINDIA DotLAN Forum Manager
> [Unwindia](https://github.com/GSH-LAN/Unwindia)'s DotLAN integration for communicate with forum threads according to tournament status
---

This service reads events with match informations from the messagequeue and publishes forum threads and comments to Dotlan. 
The published comments contains the match informations parsed into a message template using go's [text/template](https://pkg.go.dev/text/template) package.
