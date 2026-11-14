// Package domain holds core types shared across the backend: Mission,
// Astronaut, Measurement, the indicator catalog (code, unit, body system,
// related spaceflight hazard) and the Status enum (nominal|caution|critical).
//
// No I/O here: no database, no HTTP. Other packages depend on domain,
// domain depends on nothing inside the module.
package domain
