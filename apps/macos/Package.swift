// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "AlludeMacOS",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .executable(name: "AlludeMacOS", targets: ["AlludeMacOS"])
    ],
    targets: [
        .executableTarget(
            name: "AlludeMacOS",
            path: "Sources"
        )
    ]
)
