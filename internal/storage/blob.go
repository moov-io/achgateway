package storage

// TODO(adam): There might be a problem with implementing m.isolateMergableDir()
// with blob storage. Those buckets don't typically have "paths" (directories) and
// can't move them in an atomic fashion.

// TODO(adam): There might be a problem supporting 'Glob(pattern string) ([]string, error)'
// with blob storage.
