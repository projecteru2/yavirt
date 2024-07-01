package terrors

import "github.com/cockroachdb/errors"

// IsVirtLinkRouteExistsErr .
func IsVirtLinkRouteExistsErr(err error) bool {
	return errors.Is(err, ErrVirtLinkRouteExists)
}

// IsVirtLinkNotExistsErr .
func IsVirtLinkNotExistsErr(err error) bool {
	return errors.Is(err, ErrVirtLinkNotExists)
}

// IsVirtLinkAddrExistsErr .
func IsVirtLinkAddrExistsErr(err error) bool {
	return errors.Is(err, ErrVirtLinkAddrExists)
}

// IsCalicoEndpointNotExistsErr .
func IsCalicoEndpointNotExistsErr(err error) bool {
	return errors.Is(err, ErrCalicoEndpointNotExists)
}

// IsIPv4IsNetworkNumberErr .
func IsIPv4IsNetworkNumberErr(err error) bool {
	return errors.Is(err, ErrIPv4IsNetworkNumber)
}

// IsIPv4IsBroadcastErr .
func IsIPv4IsBroadcastErr(err error) bool {
	return errors.Is(err, ErrIPv4IsBroadcastAddr)
}

// IsETCDServerTimedOutErr .
func IsETCDServerTimedOutErr(err error) bool {
	return err.Error() == "etcdserver: request timed out"
}

// IsKeyNotExistsErr .
func IsKeyNotExistsErr(err error) bool {
	return errors.Is(err, ErrKeyNotExists)
}

// IsDomainNotExistsErr .
func IsDomainNotExistsErr(err error) bool {
	return errors.Is(err, ErrDomainNotExists)
}
