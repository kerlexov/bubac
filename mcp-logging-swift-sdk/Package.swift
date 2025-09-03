// swift-tools-version: 5.7
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "mcp-logging-swift-sdk",
    platforms: [
        .iOS(.v12),
        .macOS(.v10_14),
        .watchOS(.v5),
        .tvOS(.v12)
    ],
    products: [
        .library(
            name: "MCPLogging",
            targets: ["MCPLogging"]
        ),
    ],
    dependencies: [
        // No external dependencies for core functionality
    ],
    targets: [
        .target(
            name: "MCPLogging",
            dependencies: [],
            path: "Sources/MCPLogging"
        ),
        .target(
            name: "MCPLoggingSystem",
            dependencies: ["MCPLogging"],
            path: "Sources/MCPLoggingSystem"
        ),
        .testTarget(
            name: "MCPLoggingTests",
            dependencies: ["MCPLogging"],
            path: "Tests/MCPLoggingTests"
        ),
        .testTarget(
            name: "MCPLoggingIntegrationTests",
            dependencies: ["MCPLogging", "MCPLoggingSystem"],
            path: "Tests/MCPLoggingIntegrationTests"
        ),
    ]
)