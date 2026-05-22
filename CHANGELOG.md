# Changelog

## [1.8.1](https://github.com/jjack/grubstation/compare/v1.8.0...v1.8.1) (2026-05-22)


### Bug Fixes

* removing useless import from test ([bad4795](https://github.com/jjack/grubstation/commit/bad47950251206810db4068e978c546284ecba02))

## [1.8.0](https://github.com/jjack/grubstation/compare/v1.7.1...v1.8.0) (2026-05-21)


### Features

* adding config previews to dry-run; moving dry-run to a flag; fixing some linux permissions issues ([374621a](https://github.com/jjack/grubstation/commit/374621a8734ea324721229cd7373a70ec7e2cc49))
* generating hostname for windows ([c7662f5](https://github.com/jjack/grubstation/commit/c7662f5300879ecbb984de6471335974f89d1f2b))


### Bug Fixes

* back to zeroconf for mdns discovery. this now works on windows for realsies ([8100a69](https://github.com/jjack/grubstation/commit/8100a69e243fbb7676a75f8b964d8ef3c85f3adc))
* removing unnecessary condition for .msi release ([5e42d50](https://github.com/jjack/grubstation/commit/5e42d500bfb26d723498327ab161f8fcae21e016))

## [1.7.1](https://github.com/jjack/grubstation/compare/v1.7.0...v1.7.1) (2026-05-19)


### Bug Fixes

* ensuring "config init -o config.yaml" doesn't require sudo access. also fleshing out sample config files ([41020c5](https://github.com/jjack/grubstation/commit/41020c54351ff313a579f35afcc89cafb6114b04))
* making sure that "service status" uses the /status endpoint, not /healthcheck ([18fcb57](https://github.com/jjack/grubstation/commit/18fcb57de9e7d5445226b90921464ee89502fc46))

## [1.7.0](https://github.com/jjack/grubstation/compare/v1.6.1...v1.7.0) (2026-05-18)


### Features

* adding "setup --apply" to let you apply your config changes; added warning when wait_time_seconds is drifting from config ([d5d83a9](https://github.com/jjack/grubstation/commit/d5d83a9170d1dbf7e827c83894cfecd610803723))
* adding a debug log dump in caes of install error ([1af569b](https://github.com/jjack/grubstation/commit/1af569b4f860e3f0723a67acfca14b16bb5bf49f))
* using dnssd instead of mdns because it works much better with mdns reflection on windows ([668b524](https://github.com/jjack/grubstation/commit/668b52464c6a494cac8d85e068b5bfbabc9d9f05))

## [1.6.0](https://github.com/jjack/grubstation/compare/v1.5.0...v1.6.0) (2026-05-18)


### Features

* removing host from ha on uninstall ([c235e06](https://github.com/jjack/grubstation/commit/c235e0661f3a7dd9b0967a94fbac55241be3ba5e))
* validating homeassistant url before continuing ([79dea81](https://github.com/jjack/grubstation/commit/79dea81e686fa13ffac758de4f4bd0fa0584d394))


### Bug Fixes

* "serve" not "daemon" ([deacba7](https://github.com/jjack/grubstation/commit/deacba792d74d93767b13d4d3aedae40bd40c595))
* adding reinstall option to windows and moving user elevation to the .wsx so things are less weird ([eb69eb1](https://github.com/jjack/grubstation/commit/eb69eb146b0040e527c896f94247b4d63f35197e))
* elevating privileges on windows so this can actually install ([88cd458](https://github.com/jjack/grubstation/commit/88cd4586dab511387189838cc978d700dc4fbe69))
* ensuring proper config path dirname ([3b921e4](https://github.com/jjack/grubstation/commit/3b921e4e431636f8d31c2564dbf9ad802e552f50))
* getting rid of bad ascii escaping in windows installer ([48241e2](https://github.com/jjack/grubstation/commit/48241e241e8d507a68b16868ebce995b951c1948))
* hardening the mdns discovery to work better with mdns reflection across vlans. ([0ed0f05](https://github.com/jjack/grubstation/commit/0ed0f058b93a309d277ac2f31364c4aa4e8d2617))
* hardening the mdns discovery to work better with mdns reflection across vlans. ([d2f5bc0](https://github.com/jjack/grubstation/commit/d2f5bc0fa729de34f8d3a39e8f3807724a25ef99))
* moving user elevation in windows to .wsx ([d5b80be](https://github.com/jjack/grubstation/commit/d5b80bee2ed3b9f62e375659b5088aadeb50de44))
* removing config file when uninstalling ([fb3f34d](https://github.com/jjack/grubstation/commit/fb3f34d132d1c178f09f4048c5d15c15d3dddafe))
* showing vlan subnet broadcast for installer even if just a hostname or fqdn is selected ([c4346b9](https://github.com/jjack/grubstation/commit/c4346b9231eb1fd230341da6f6376828f733208e))
* windows installer now has install path as an option. plus uninstaller removes config files. ([455152f](https://github.com/jjack/grubstation/commit/455152ff5e3ce02f6cc76569af0cabb2daba17a1))

## [1.5.0](https://github.com/jjack/grubstation/compare/v1.4.0...v1.5.0) (2026-05-16)


### Features

* /shutdown now reports actual shutdown errors ([49464b7](https://github.com/jjack/grubstation/commit/49464b714fb2d1736b3159c4c6161ff43f48be12))
* adding /status endpoing instead of basic /healthcheck ([5939984](https://github.com/jjack/grubstation/commit/5939984a19c229cab24be71f36cc8fa51f5d2398))
* allowing discovery of multiple home assistant instances and addresses ([8d4b793](https://github.com/jjack/grubstation/commit/8d4b79342e5cf0c57a0de08b23502672d3f49d33))
* supporting multiple home assistant instances, and different URLs for the agent and for grub so we can have https support ([c6e3b1b](https://github.com/jjack/grubstation/commit/c6e3b1b1822d30d93f871ce107850dca3c950ffd))


### Bug Fixes

* adding agent_port to payload ([e57879a](https://github.com/jjack/grubstation/commit/e57879a9dbf69834900dffe96ff66bb60e3eb130))
* correcting grubstation service name ([a773e9b](https://github.com/jjack/grubstation/commit/a773e9bd9800f23fd26dd90f4d8edb3c75c16cae))
* injecting ha client into reporter instead of rebuilding it ([b4dfc6b](https://github.com/jjack/grubstation/commit/b4dfc6b735a2ea0882c31de317aa0274273f8cb1))
* removing token from log output ([45a7ff3](https://github.com/jjack/grubstation/commit/45a7ff37e22805152a3f3e61cc50b6ce01877313))
* removing unused import ([fd5101f](https://github.com/jjack/grubstation/commit/fd5101f9aa4ad87168a306859eae2bd2a8902050))
* renaming cli to grubstation ([5ad1a39](https://github.com/jjack/grubstation/commit/5ad1a398c9025b69f2d45702dd67df0fe02abcd3))
* splitting up logic for windows vs linux physical network interface detection ([276a8fb](https://github.com/jjack/grubstation/commit/276a8fb4a3258439823f30c783e3383bfdb7a059))
* updating payload to have correct keys ([1df1fe6](https://github.com/jjack/grubstation/commit/1df1fe63f8547eee8eabf37950136d5106790bb8))
* updating payload to have correct keys ([39367e2](https://github.com/jjack/grubstation/commit/39367e27baeaf79fb90aab727e56e90577cf9386))

## [1.4.0](https://github.com/jjack/grub-os-reporter/compare/v1.3.2...v1.4.0) (2026-05-07)


### Features

* auto-running "push" after successful setup ([847914a](https://github.com/jjack/grub-os-reporter/commit/847914aa49843b9a78eb025386cc17c4eec3bee0))


### Bug Fixes

* added flag strings to config ([a9a5dc7](https://github.com/jjack/grub-os-reporter/commit/a9a5dc75cd354d9f7c3222bb3a3c489fd5932eb8))
* ensuring directory exists before saving config.yaml ([1f71fea](https://github.com/jjack/grub-os-reporter/commit/1f71fea1d5372fa47396d89137a465b2f3549eca))
* more robust handling of grub networking ([72b14d6](https://github.com/jjack/grub-os-reporter/commit/72b14d6f2fc95b27258895ea9c4baad857903ec0))
* removing my hard-coded test mac address from the grub template ([32ece9c](https://github.com/jjack/grub-os-reporter/commit/32ece9c78d88bda5d8d5fa7615ac09b88666da47))
* securing newly created config file ([1d78da0](https://github.com/jjack/grub-os-reporter/commit/1d78da059f22a9f0c7b0d4ec89529e8f2c17334c))
* using the final, real path for config files /etc/remote-boot-agent/ instead of $pwd ([bfdba77](https://github.com/jjack/grub-os-reporter/commit/bfdba774a98f13e06a0a3cb5b23616322ba3459c))

## 1.3.2 (2026-05-06)


### Features

* add config generator ([#2](https://github.com/jjack/remote-boot-agent/issues/2)) ([e1b2486](https://github.com/jjack/remote-boot-agent/commit/e1b2486c98f7a5e90518ec1e20d29356ed98e793))
* adding "install" command ([5b95933](https://github.com/jjack/remote-boot-agent/commit/5b959333fbdfad0ef1712f937894ee0656fb12e2))
* adding broadcast address and port to config ([ff5bb93](https://github.com/jjack/remote-boot-agent/commit/ff5bb932b8555227616819bfef7f4d8ac4b69658))
* adding timeout for http client ([84d8e66](https://github.com/jjack/remote-boot-agent/commit/84d8e66a098240ce8bb2608d7043b7c776aa0d07))
* adding timeout for http client ([7a7359f](https://github.com/jjack/remote-boot-agent/commit/7a7359f3082c1100309f133d5c8c55b230b05e49))
* allowing custom webhook_ids for security ([b46975b](https://github.com/jjack/remote-boot-agent/commit/b46975ba8d337ca365e138db8dbdd01e9ef6377e))
* allowing custom webhook_ids for security ([8c97cc3](https://github.com/jjack/remote-boot-agent/commit/8c97cc3aa1947cde6048cf21261ce85c26536ce5))
* autodiscovery of init and bootloader, also parsing grub config ([5860459](https://github.com/jjack/remote-boot-agent/commit/58604598af79145a7341e12d748440c720b49eec))
* autodiscovery of init and bootloader, also parsing grub config ([169edcc](https://github.com/jjack/remote-boot-agent/commit/169edcc68e621248c061249749faba2f63fe42c6))
* autodiscovery of init and bootloader, also parsing grub config ([310350c](https://github.com/jjack/remote-boot-agent/commit/310350c9a4215000e500ad42944441d6f972e41d))
* autodiscovery of init and bootloader, also parsing grub config ([6d0edb5](https://github.com/jjack/remote-boot-agent/commit/6d0edb595e3d4a1954385357d8b45fbbcfd5270b))
* changing config to better match wake on lan ([08192cc](https://github.com/jjack/remote-boot-agent/commit/08192cc1d1437406f8152d14a4e205f86f829093))
* config file parsing ([ab0ff0b](https://github.com/jjack/remote-boot-agent/commit/ab0ff0be8b8babeb7628469472e95c6f70f810ee))
* config file parsing ([da5af6f](https://github.com/jjack/remote-boot-agent/commit/da5af6ffc7dddb718596776e464e3c17416db014))
* get-available-oses to print what youve got ([3cea631](https://github.com/jjack/remote-boot-agent/commit/3cea631cd9be57162fb0e27f6a61b9b7215ccaae))
* get-available-oses to print what youve got ([2cad4ac](https://github.com/jjack/remote-boot-agent/commit/2cad4acc2d8653789cc4dd22285684364b606789))
* initial (generated) cli layout ([8ec09c1](https://github.com/jjack/remote-boot-agent/commit/8ec09c1fcd6d05ddc02db6fc6458583c53a14639))
* initial (generated) cli layout ([7cbd9f3](https://github.com/jjack/remote-boot-agent/commit/7cbd9f36958f2e277faab106f6329dd8875aafc5))
* initial ansible config ([7edaf70](https://github.com/jjack/remote-boot-agent/commit/7edaf7099bd069389a961c64963bb8e581c47073))
* initial ansible config ([1cc0b27](https://github.com/jjack/remote-boot-agent/commit/1cc0b2796edb49f4bb1a5448855cf0c671860c09))
* making bootloader config a little more generic so that it can work with other bootloaders in the future ([c4344fc](https://github.com/jjack/remote-boot-agent/commit/c4344fc4a65e70e9d06d7b84beea0d90859bb97e))
* optional SetupWarning to display additional info post setup ([477c98a](https://github.com/jjack/remote-boot-agent/commit/477c98a1691715ac30e1adbd4661206a62d37cfe))
* switching to actual logging ([e864fd9](https://github.com/jjack/remote-boot-agent/commit/e864fd90fcf07de87a2b4d62d82a2f3e10bf169d))
* switching to actual logging ([d8948e7](https://github.com/jjack/remote-boot-agent/commit/d8948e79c91c3ec20858557466f8a31790c9ccd0))


### Bug Fixes

* "config generate --path" now saves to the proper path ([e3a7e31](https://github.com/jjack/remote-boot-agent/commit/e3a7e310c2d8af6777363363404bafe44cdb55bf))
* actually validating the config during config validation ([cb85e6b](https://github.com/jjack/remote-boot-agent/commit/cb85e6b34fd2974946c4aab1560a6ba10fcc79df))
* adding all the rest of the cli options to the actual cli ([b42db58](https://github.com/jjack/remote-boot-agent/commit/b42db588fb0467b04ef7188db168981d658d9305))
* adding all the rest of the cli options to the actual cli ([f598b8f](https://github.com/jjack/remote-boot-agent/commit/f598b8f5363e426a3a9e60b74a8fc53b5f44dfd2))
* adding config generation for bootloader and init system ([7234984](https://github.com/jjack/remote-boot-agent/commit/723498474b377428ab09ad8c7f2adc067f620c74))
* adding error-handling for BindPFlag ([98acfb8](https://github.com/jjack/remote-boot-agent/commit/98acfb84753a5f176f4cee90be5ecc639a625bd5))
* adding error-handling for BindPFlag ([a43f155](https://github.com/jjack/remote-boot-agent/commit/a43f1557e5dc51d1a7fbf8f042e436cd7d87eb94))
* adding initsystem and bootloader to config generation ([6e964e1](https://github.com/jjack/remote-boot-agent/commit/6e964e1326801e9f53f3cb9cf02aa2924ed106cf))
* adding webhook id as a token param to match home assistant ([9e38232](https://github.com/jjack/remote-boot-agent/commit/9e382323b0ea7ccc45fb69c6c95e52a477d88ecb))
* allowing bootloader config overrides on the cli ([7100d03](https://github.com/jjack/remote-boot-agent/commit/7100d039439ce1edef1726597134ac36b882a5a9))
* allowing bootloader config overrides on the cli ([66dbbdd](https://github.com/jjack/remote-boot-agent/commit/66dbbddf054441f4220705964acd0344b387258b))
* allowing overrides for grub config path ([a14e61a](https://github.com/jjack/remote-boot-agent/commit/a14e61a20e77f7cf310ab909897372cdd0cefb38))
* allowing overrides for grub config path ([82408c7](https://github.com/jjack/remote-boot-agent/commit/82408c7cb5cb7e55cc1063f071ec73cab2f51dc9))
* better handling of brace counting for grub bootloader ([3062876](https://github.com/jjack/remote-boot-agent/commit/3062876ac1203fbe5954497f9e812343bd5bf0b7))
* better handling of grub networking ([5da4cba](https://github.com/jjack/remote-boot-agent/commit/5da4cba9ac05519c0b709a178aa504f8b610681a))
* better handling of permission issues when reading grub config ([8c4fedd](https://github.com/jjack/remote-boot-agent/commit/8c4fedd77e8b10f5035f9cf18cad9de840e6cd06))
* better parsing of grub config and handling of submenus ([b5a4795](https://github.com/jjack/remote-boot-agent/commit/b5a47958902bf2135bb649f7d58f76f5b0fdf426))
* can now cancel home assistant discovery ([0eb6628](https://github.com/jjack/remote-boot-agent/commit/0eb66285659d628b4b0f50a0a7dd6ce14ac2b424))
* closing grub file before update-grub runs ([05d786f](https://github.com/jjack/remote-boot-agent/commit/05d786f0cf0a7d1a148297c87d27f1c6a1e43aaa))
* correcting logic on physical interface path checking ([c145f99](https://github.com/jjack/remote-boot-agent/commit/c145f99f963d4d68d4d663065a2bce73e656762f))
* ensuring test files are closed ([dd7a1a6](https://github.com/jjack/remote-boot-agent/commit/dd7a1a6f0e5a0b41934ffe52fbec1c4e1e506aa9))
* fixing  config file handling ([8c19e77](https://github.com/jjack/remote-boot-agent/commit/8c19e77afa9dfc883aa524dfd009a2276a707ef7))
* fixing host/server mismatches ([cb1190f](https://github.com/jjack/remote-boot-agent/commit/cb1190f69499aebb1da5c1a768cd3a1f63b3b3ef))
* fixing potential memory leak if channel doesn't get closed ([c6a941b](https://github.com/jjack/remote-boot-agent/commit/c6a941b379d3af24d377803983b2195f7c8b3723))
* getting/pushing oses. cleaning up config files ([0da0ecf](https://github.com/jjack/remote-boot-agent/commit/0da0ecf6a71eeeeb07190f403cb63c858f29d96d))
* getting/pushing oses. cleaning up config files ([3f26e35](https://github.com/jjack/remote-boot-agent/commit/3f26e355bf9d1195c7482bad319e39ee87fc5cfa))
* grub_path =&gt; grubPath ([a71d687](https://github.com/jjack/remote-boot-agent/commit/a71d687c7777753763a859daebe0fa49af0f8168))
* grub_path =&gt; grubPath ([eb86875](https://github.com/jjack/remote-boot-agent/commit/eb86875459da17a2ed7e47ff52d20e8cf863d2f8))
* making sure that config generation can have its context canceled ([c48a612](https://github.com/jjack/remote-boot-agent/commit/c48a612b1ce3fdf6deb859b3e6e16783f600acff))
* moving logging to debug to reduce noise ([b3be744](https://github.com/jjack/remote-boot-agent/commit/b3be74476c6cd197f11ee559a02954a2fc1581b7))
* moving logging to debug to reduce noise ([e55adae](https://github.com/jjack/remote-boot-agent/commit/e55adaede68f3601eac5093c8a170dffe0bb4efd))
* no longer building with example.go ([d895bf4](https://github.com/jjack/remote-boot-agent/commit/d895bf41728c0c507f2240de15a95a5c47912068))
* normalizing found mac addresses ([1fc8f44](https://github.com/jjack/remote-boot-agent/commit/1fc8f44f129b428ed5d588077b21548ff774f38a))
* removing cancel from the func to rely on the main defer cancel ([18ee0f2](https://github.com/jjack/remote-boot-agent/commit/18ee0f2b20dd6ae1bb01c1c88353dc97d602b40d))
* removing hardcoded home assistant url to let users put their own in ([31ba548](https://github.com/jjack/remote-boot-agent/commit/31ba5487d89576cd0bf6b2aa701f808e0e145707))
* removing tokens because everything is unauthenticated ([c8322df](https://github.com/jjack/remote-boot-agent/commit/c8322dfe0633dc4adf925fd35a2e708e3f28c516))
* removing tokens because everything is unauthenticated ([4d96c41](https://github.com/jjack/remote-boot-agent/commit/4d96c412c1654510dffe59004dcfb42f71e64a2a))
* removing useless error handling ([26aa5ea](https://github.com/jjack/remote-boot-agent/commit/26aa5ea7e325ed3c151284ac9ffb3c3491170d3f))
* removing useless error handling ([5a15aff](https://github.com/jjack/remote-boot-agent/commit/5a15aff4a306f21d3692d1925c2b052fda844127))
* suppressing some warnings for unchecked return values ([5a8a741](https://github.com/jjack/remote-boot-agent/commit/5a8a74198cb7e8e2363f36769d536d540b074782))
* suppressing some warnings for unchecked return values ([589343e](https://github.com/jjack/remote-boot-agent/commit/589343e5972aa2541d8a5911a665d41ddfcaea53))
* switching to os.UserHomeDir() beacuse Viper doesn't expand $HOME ([5853a5e](https://github.com/jjack/remote-boot-agent/commit/5853a5e3db1d7ca4f2d651c757b305d6a38cc15e))
* switching to os.UserHomeDir() beacuse Viper doesn't expand $HOME ([15e46b5](https://github.com/jjack/remote-boot-agent/commit/15e46b54ffa310fc24ede57e1824bdb1d03b0b69))
* updating bootloader and network stuff to accept contexes ([3eff4fa](https://github.com/jjack/remote-boot-agent/commit/3eff4faec1da99e962ea2881d1cd559a358f6e49))
* updating gorelease ([25efbd5](https://github.com/jjack/remote-boot-agent/commit/25efbd5ecac673e4bd84faf9b52e175f39c538e6))
* using a custom buffer to support bootloaders &gt; 64kb ([8a65d81](https://github.com/jjack/remote-boot-agent/commit/8a65d81d20cdba64b8183e8316bc161bfd8f7ad4))
* using mac address instead of hostname for queries ([a95db06](https://github.com/jjack/remote-boot-agent/commit/a95db06969a56746e59f014574d8c4523adf665d))
* using mac address instead of hostname for queries ([4497f3d](https://github.com/jjack/remote-boot-agent/commit/4497f3d7bbf37e4555a463331fda35fe2343355d))
* using mac instead of mac_address to match HA's format ([e4032ad](https://github.com/jjack/remote-boot-agent/commit/e4032ad0cd712f15a35e53d30cd375dfa2d6ee8a))
* using mac instead of mac_address to match HA's format ([f2a8ed0](https://github.com/jjack/remote-boot-agent/commit/f2a8ed0fc74f55cf73ca11761d71ec3668f4ac4e))
* using println for prettier stdout ([1c075e8](https://github.com/jjack/remote-boot-agent/commit/1c075e8a6302da914c55cb5e241dc0ba54669085))
* using println for prettier stdout ([c439c64](https://github.com/jjack/remote-boot-agent/commit/c439c64d643ef076b8079a902f96f98bacf31263))


### Miscellaneous Chores

* force release ([a4038f4](https://github.com/jjack/remote-boot-agent/commit/a4038f44e46098723ec4f3b3c5ee2b68e557a28c))

## [1.3.1](https://github.com/jjack/remote-boot-agent/compare/v1.3.0...v1.3.1) (2026-04-29)


### Bug Fixes

* better handling of grub networking ([8b393f6](https://github.com/jjack/remote-boot-agent/commit/8b393f6a27ef94938d9822822b77527db7b4671b))

## [1.3.0](https://github.com/jjack/remote-boot-agent/compare/v1.2.1...v1.3.0) (2026-04-29)


### Features

* adding broadcast address and port to config ([573b0d1](https://github.com/jjack/remote-boot-agent/commit/573b0d1e49fc7876713c4895103fb0f2c4c8cb79))


### Bug Fixes

* "config generate --path" now saves to the proper path ([cb15c63](https://github.com/jjack/remote-boot-agent/commit/cb15c630e27e29fa1d7c622d71800b7a5febfa67))
* actually validating the config during config validation ([9f63b7f](https://github.com/jjack/remote-boot-agent/commit/9f63b7f047a0051e9622648b19f901b71f706f92))
* can now cancel home assistant discovery ([d3a7964](https://github.com/jjack/remote-boot-agent/commit/d3a7964f0bbd44bbe503e8c4cc37477484e32c62))

## [1.2.1](https://github.com/jjack/remote-boot-agent/compare/v1.2.0...v1.2.1) (2026-04-25)


### Bug Fixes

* updating gorelease ([5f5a2db](https://github.com/jjack/remote-boot-agent/commit/5f5a2dbe8fb5bf44a3ecd4c32ab503eb0154aeaf))

## [1.2.0](https://github.com/jjack/remote-boot-agent/compare/v1.1.0...v1.2.0) (2026-04-25)


### Features

* adding "install" command ([89c5f80](https://github.com/jjack/remote-boot-agent/commit/89c5f808c173ba16e7fdc4ba8bf5c3d8b673cc16))


### Bug Fixes

* adding config generation for bootloader and init system ([102151d](https://github.com/jjack/remote-boot-agent/commit/102151d39638ae00948c0bbb273d12b0532f3e1c))
* adding initsystem and bootloader to config generation ([dc949a1](https://github.com/jjack/remote-boot-agent/commit/dc949a1d31bf458c699a497da52db3879d702c80))
* closing grub file before update-grub runs ([babe011](https://github.com/jjack/remote-boot-agent/commit/babe011ec1e65ca8b4178f9b21963b1a63b692cc))
* fixing  config file handling ([2590426](https://github.com/jjack/remote-boot-agent/commit/2590426a3761e672e26a3d7bc69608b033a26101))
* updating bootloader and network stuff to accept contexes ([249dcbd](https://github.com/jjack/remote-boot-agent/commit/249dcbd26ff69791277c29207cabc9076a86d3fc))

## [1.1.0](https://github.com/jjack/remote-boot-agent/compare/v1.0.0...v1.1.0) (2026-04-25)


### Features

* add config generator ([#2](https://github.com/jjack/remote-boot-agent/issues/2)) ([9ee8f1d](https://github.com/jjack/remote-boot-agent/commit/9ee8f1d946704027fcaca7252c86a78714619971))
* making bootloader config a little more generic so that it can work with other bootloaders in the future ([bc59f73](https://github.com/jjack/remote-boot-agent/commit/bc59f7398cd63c35a161162d30ce0b07550bf840))


### Bug Fixes

* better handling of brace counting for grub bootloader ([5c7dd95](https://github.com/jjack/remote-boot-agent/commit/5c7dd952a4ff928cdb446fb66d649d41c45a5987))
* better handling of permission issues when reading grub config ([e6702a1](https://github.com/jjack/remote-boot-agent/commit/e6702a16cfaf97dc8dab210dbb4e18a6967b30bd))
* better parsing of grub config and handling of submenus ([721045a](https://github.com/jjack/remote-boot-agent/commit/721045a3a9ccc5eab3c60a1d0eabbf4315a690cb))
* fixing potential memory leak if channel doesn't get closed ([aec7946](https://github.com/jjack/remote-boot-agent/commit/aec7946f503658cec13d5873d37487c4728b7ec6))
* normalizing found mac addresses ([167be81](https://github.com/jjack/remote-boot-agent/commit/167be81a33de153afb51bbc60cd2a3589a3c8b27))
* removing hardcoded home assistant url to let users put their own in ([e7d8ce3](https://github.com/jjack/remote-boot-agent/commit/e7d8ce3c0dccfe47270911c0b33adc56104f8d33))
* using a custom buffer to support bootloaders &gt; 64kb ([b0de10c](https://github.com/jjack/remote-boot-agent/commit/b0de10cdb490ac1764a341c9b11a236b97daa721))

## 1.0.0 (2026-04-17)


### Features

* adding timeout for http client ([84d8e66](https://github.com/jjack/remote-boot-agent/commit/84d8e66a098240ce8bb2608d7043b7c776aa0d07))
* adding timeout for http client ([7a7359f](https://github.com/jjack/remote-boot-agent/commit/7a7359f3082c1100309f133d5c8c55b230b05e49))
* allowing custom webhook_ids for security ([b46975b](https://github.com/jjack/remote-boot-agent/commit/b46975ba8d337ca365e138db8dbdd01e9ef6377e))
* allowing custom webhook_ids for security ([8c97cc3](https://github.com/jjack/remote-boot-agent/commit/8c97cc3aa1947cde6048cf21261ce85c26536ce5))
* autodiscovery of init and bootloader, also parsing grub config ([5860459](https://github.com/jjack/remote-boot-agent/commit/58604598af79145a7341e12d748440c720b49eec))
* autodiscovery of init and bootloader, also parsing grub config ([169edcc](https://github.com/jjack/remote-boot-agent/commit/169edcc68e621248c061249749faba2f63fe42c6))
* autodiscovery of init and bootloader, also parsing grub config ([310350c](https://github.com/jjack/remote-boot-agent/commit/310350c9a4215000e500ad42944441d6f972e41d))
* autodiscovery of init and bootloader, also parsing grub config ([6d0edb5](https://github.com/jjack/remote-boot-agent/commit/6d0edb595e3d4a1954385357d8b45fbbcfd5270b))
* config file parsing ([ab0ff0b](https://github.com/jjack/remote-boot-agent/commit/ab0ff0be8b8babeb7628469472e95c6f70f810ee))
* config file parsing ([da5af6f](https://github.com/jjack/remote-boot-agent/commit/da5af6ffc7dddb718596776e464e3c17416db014))
* get-available-oses to print what youve got ([3cea631](https://github.com/jjack/remote-boot-agent/commit/3cea631cd9be57162fb0e27f6a61b9b7215ccaae))
* get-available-oses to print what youve got ([2cad4ac](https://github.com/jjack/remote-boot-agent/commit/2cad4acc2d8653789cc4dd22285684364b606789))
* initial (generated) cli layout ([8ec09c1](https://github.com/jjack/remote-boot-agent/commit/8ec09c1fcd6d05ddc02db6fc6458583c53a14639))
* initial (generated) cli layout ([7cbd9f3](https://github.com/jjack/remote-boot-agent/commit/7cbd9f36958f2e277faab106f6329dd8875aafc5))
* initial ansible config ([7edaf70](https://github.com/jjack/remote-boot-agent/commit/7edaf7099bd069389a961c64963bb8e581c47073))
* initial ansible config ([1cc0b27](https://github.com/jjack/remote-boot-agent/commit/1cc0b2796edb49f4bb1a5448855cf0c671860c09))
* switching to actual logging ([e864fd9](https://github.com/jjack/remote-boot-agent/commit/e864fd90fcf07de87a2b4d62d82a2f3e10bf169d))
* switching to actual logging ([d8948e7](https://github.com/jjack/remote-boot-agent/commit/d8948e79c91c3ec20858557466f8a31790c9ccd0))


### Bug Fixes

* adding all the rest of the cli options to the actual cli ([b42db58](https://github.com/jjack/remote-boot-agent/commit/b42db588fb0467b04ef7188db168981d658d9305))
* adding all the rest of the cli options to the actual cli ([f598b8f](https://github.com/jjack/remote-boot-agent/commit/f598b8f5363e426a3a9e60b74a8fc53b5f44dfd2))
* adding error-handling for BindPFlag ([98acfb8](https://github.com/jjack/remote-boot-agent/commit/98acfb84753a5f176f4cee90be5ecc639a625bd5))
* adding error-handling for BindPFlag ([a43f155](https://github.com/jjack/remote-boot-agent/commit/a43f1557e5dc51d1a7fbf8f042e436cd7d87eb94))
* allowing bootloader config overrides on the cli ([7100d03](https://github.com/jjack/remote-boot-agent/commit/7100d039439ce1edef1726597134ac36b882a5a9))
* allowing bootloader config overrides on the cli ([66dbbdd](https://github.com/jjack/remote-boot-agent/commit/66dbbddf054441f4220705964acd0344b387258b))
* allowing overrides for grub config path ([a14e61a](https://github.com/jjack/remote-boot-agent/commit/a14e61a20e77f7cf310ab909897372cdd0cefb38))
* allowing overrides for grub config path ([82408c7](https://github.com/jjack/remote-boot-agent/commit/82408c7cb5cb7e55cc1063f071ec73cab2f51dc9))
* getting/pushing oses. cleaning up config files ([0da0ecf](https://github.com/jjack/remote-boot-agent/commit/0da0ecf6a71eeeeb07190f403cb63c858f29d96d))
* getting/pushing oses. cleaning up config files ([3f26e35](https://github.com/jjack/remote-boot-agent/commit/3f26e355bf9d1195c7482bad319e39ee87fc5cfa))
* grub_path =&gt; grubPath ([a71d687](https://github.com/jjack/remote-boot-agent/commit/a71d687c7777753763a859daebe0fa49af0f8168))
* grub_path =&gt; grubPath ([eb86875](https://github.com/jjack/remote-boot-agent/commit/eb86875459da17a2ed7e47ff52d20e8cf863d2f8))
* moving logging to debug to reduce noise ([b3be744](https://github.com/jjack/remote-boot-agent/commit/b3be74476c6cd197f11ee559a02954a2fc1581b7))
* moving logging to debug to reduce noise ([e55adae](https://github.com/jjack/remote-boot-agent/commit/e55adaede68f3601eac5093c8a170dffe0bb4efd))
* removing tokens because everything is unauthenticated ([c8322df](https://github.com/jjack/remote-boot-agent/commit/c8322dfe0633dc4adf925fd35a2e708e3f28c516))
* removing tokens because everything is unauthenticated ([4d96c41](https://github.com/jjack/remote-boot-agent/commit/4d96c412c1654510dffe59004dcfb42f71e64a2a))
* removing useless error handling ([26aa5ea](https://github.com/jjack/remote-boot-agent/commit/26aa5ea7e325ed3c151284ac9ffb3c3491170d3f))
* removing useless error handling ([5a15aff](https://github.com/jjack/remote-boot-agent/commit/5a15aff4a306f21d3692d1925c2b052fda844127))
* suppressing some warnings for unchecked return values ([5a8a741](https://github.com/jjack/remote-boot-agent/commit/5a8a74198cb7e8e2363f36769d536d540b074782))
* suppressing some warnings for unchecked return values ([589343e](https://github.com/jjack/remote-boot-agent/commit/589343e5972aa2541d8a5911a665d41ddfcaea53))
* switching to os.UserHomeDir() beacuse Viper doesn't expand $HOME ([5853a5e](https://github.com/jjack/remote-boot-agent/commit/5853a5e3db1d7ca4f2d651c757b305d6a38cc15e))
* switching to os.UserHomeDir() beacuse Viper doesn't expand $HOME ([15e46b5](https://github.com/jjack/remote-boot-agent/commit/15e46b54ffa310fc24ede57e1824bdb1d03b0b69))
* using mac address instead of hostname for queries ([a95db06](https://github.com/jjack/remote-boot-agent/commit/a95db06969a56746e59f014574d8c4523adf665d))
* using mac address instead of hostname for queries ([4497f3d](https://github.com/jjack/remote-boot-agent/commit/4497f3d7bbf37e4555a463331fda35fe2343355d))
* using mac instead of mac_address to match HA's format ([e4032ad](https://github.com/jjack/remote-boot-agent/commit/e4032ad0cd712f15a35e53d30cd375dfa2d6ee8a))
* using mac instead of mac_address to match HA's format ([f2a8ed0](https://github.com/jjack/remote-boot-agent/commit/f2a8ed0fc74f55cf73ca11761d71ec3668f4ac4e))
* using println for prettier stdout ([1c075e8](https://github.com/jjack/remote-boot-agent/commit/1c075e8a6302da914c55cb5e241dc0ba54669085))
* using println for prettier stdout ([c439c64](https://github.com/jjack/remote-boot-agent/commit/c439c64d643ef076b8079a902f96f98bacf31263))
