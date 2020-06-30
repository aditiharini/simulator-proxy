# simulator-proxy

Current plan:
- Finish writing trace emulator in proxy : 
This will make writing de-duplication logic and taking measurements a lot easier. I can just instrument the proxy to do this. 
- Take measurements:
Using sender program, send a bunch of packets to the receiver. Determine which path first packet to arrive took. 
