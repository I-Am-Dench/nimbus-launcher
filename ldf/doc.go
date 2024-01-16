// This implementation is specified through the use of the lu_packet_structs doc,
// and the DarkflameUniverse server implementation.
//
// > lu_packet_structs: https://docs.google.com/document/d/1v9GB1gNwO0C81Rhd4imbaLN7z-R0zpK5sYJMbxPP3Kc/edit#heading=h.mfywx277s2j9
//
// > LDFFormat.h: https://github.com/DarkflameUniverse/DarkflameServer/blob/main/dCommon/LDFFormat.h
//
// This implementation is incomplete as some types are missing which are not necessary to the launcher's operation.
// If the rest of the types were to be added, this package would most likely be moved into its own module.
//
// For purposes of the launcher, value type 0 is marshalled and unmarshalled as utf-8 rather than utf-16.
package ldf
