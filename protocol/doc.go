/*
Package protocol is package for Aries protocol processors. Protocol processors
implement the actual protocol state transitions. The protocol specific message
implementations are located in std package. The PSM is in agent/psm which
includes the representatives for protocol-based state variables.

These processors include the dynamic logic of the protocol state machines. When
new implementations of the protocols are needed the are put here.
*/
package protocol
