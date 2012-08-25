Level 7 notes (level07)
=======================

This level involves a piece of software known as WaffleCopter, a waffle delivery service. Mmm, tasty.

What's the vulnerability?
-------------------------

To put it simply, a hash length extension attack. This attack allows you to append arbitrary data to the end of something and then generate a new hash for it.

If you have SECRET_SECRET_aaaaaa with the hash ed006d09e8e757d67b732a2104c0baed8460e3dc, then you can make it SECRET_SECRET_aaaaaa(padding here)bbbbbb with the hash 0901d3b98322c3c486ad1ef03f6fe5ab69f10ba9 *without knowing the SECRET_SECRET_ bit*.

Thanks to http://www.vnsecurity.net/t/length-extension-attack/ for the code, by the way - I <strike>stole</strike> used their source code to carry out this attack.

Any more details?
-----------------

Not really - it's fairly straightforward once you know what the attack needs to be. You just exploit the publically available logs that WaffleCopter makes available to you by changing the trailing /5 in the log URL to a /1.