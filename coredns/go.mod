module github.com/networkservicemesh/fanmerge/coredns

go 1.13

require (
	github.com/coredns/coredns v1.7.0
	github.com/networkservicemesh/fanmerge v0.0.0-20200313150119-ddef81d89163
	github.com/networkservicemesh/fanout v0.0.0-20200803070946-b663b8bd9437
)

replace github.com/networkservicemesh/fanmerge => ../
