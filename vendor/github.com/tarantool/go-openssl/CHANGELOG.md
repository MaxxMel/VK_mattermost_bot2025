# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic
Versioning](http://semver.org/spec/v2.0.0.html) except to the first release.

## [Unreleased]

### Added

### Changed

### Fixed

## [v1.1.1] - 2024-09-27

The small release include fixes for problems found by Svacer.

### Fixed

- Unchecked `X509_STORE_CTX_get_ex_data` return value (#16).

## [v1.1.0] - 2024-09-02

The release adds more bindings.

### Added

- Bindings for [DANE](https://docs.openssl.org/1.1.1/man3/SSL_CTX_dane_enable/) (#14).
- Bindings for [TLS handshake tracing](https://docs.openssl.org/master/man3/SSL_CTX_set_msg_callback/) (#14).
- Bindings for `X509_digest()` (#14).
- Bindings for `X509_verify_cert_error_string()` (#14).
- Bindings for `SSL_get_version()` (#14).

## [v1.0.0] - 2024-02-09

The first release with a number of fixes. Since `libp2p/openssl` is not
supported any more we need to support our version for usage in the Golang
connector `tarantool/go-tarantool`.

See [releases of `libp2p/openssl`](https://github.com/libp2p/go-openssl/releases)
for previous changes history.

### Added

- DialContext function (#10).

### Fixed

- Build by Golang 1.13 (#6).
- Build with OpenSSL < 1.1.1 (#7).
- Build on macOS as a static library (#8).
- Build on macOS with Apple M1 (#8).
- Random errors in the code caused by an invalid OpenSSL error handling in
  LoadPrivateKeyFromPEM, LoadPrivateKeyFromPEMWithPassword,
  LoadPrivateKeyFromDER and LoadPublicKeyFromPEM (#9).
