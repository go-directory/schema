# schema

Package schema implements various subsets of [RFC4512](https://www.rfc-editor.org/info/rfc4512/) to facilitate a thread-safe LDAP schema definition parser, storage type and interrogation platform.

This package was specifically designed for use by the [Go Directory Project](https://github.com/go-directory).

# About

Textual schema definitions parsed by this package are "digested" -- that is, tokenized, parsed and dispersed throughout myriad fast lookup tables.

This library does not store actual "definition objects". Instead, it writes all pertinent tokens to appropriate tables, optimized for _interrogative_ use. Discrete values are numbered, _not named_, and generally favor the `[]byte` type.

The package is written primarily for performance. Heavy lifting, such as super class traversal, expansion of available types and string representation, is done in advance. This makes it ideal for use in cases where performance is a top concern, such as when interacting with a storage backend (e.g.: a database).

# License

This package is released under the terms of the MIT license. See the LICENSE file in the repository root for details.

# Status

This package is undergoing heavy development and, as such, as considered EXPERIMENTAL. It should NOT be used in any production environment.
