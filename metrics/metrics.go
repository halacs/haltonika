package metrics

type TeltonikaMetricsInterface interface {
	AddSentBytes(count uint64)
	AddReceivedBytes(count uint64)
	AddSentPackages(count uint64)
	AddReceivedPackages(count uint64)
	AddMalformedPackages(count uint64)
	AddRejectedPackages(count uint64)
	AddResentPackages(count uint64)
}
