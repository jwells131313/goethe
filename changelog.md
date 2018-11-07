# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed

## [1.3.0] - 2018-11-07
### Changed
- Added API to add a destructor function to the CAR cache for destroying a value when it is removed from the cache

## [1.2.0] - 2018-10-16
### Changed
- Added new API to Lock TryWriteLock and TryReadLock which allow for timed
waiting for the lock to free up
- Added new API to Lock for seeing if a lock is read/write locked
- Added new API in new package (queues) for general Heap algorithm queue

## [1.1.0] - 2018-09-28
### Changed
- Documentation fixes
- Use go 1.11 modules for versioning
- New feature:  CAR cache!

## [1.0.0] - 2018-09-12
### Added
- Threads which have IDs called Goethe threads
- Recursive (counting) locks on Goethe threads
- Thread Local Storage on Goethe threads
- Timers that run on Goethe threads
- Thread pools that run Goethe threads
- Computable Cache
