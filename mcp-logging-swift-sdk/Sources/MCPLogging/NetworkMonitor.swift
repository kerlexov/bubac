import Foundation
import Network

/// Monitors network reachability and notifies about changes
@available(iOS 12.0, macOS 10.14, watchOS 5.0, tvOS 12.0, *)
public class NetworkMonitor {
    public enum NetworkStatus {
        case satisfied
        case unsatisfied
        case requiresConnection
    }
    
    private let monitor: NWPathMonitor
    private let queue: DispatchQueue
    private var isMonitoring = false
    
    public var statusUpdateHandler: ((NetworkStatus) -> Void)?
    public var pathUpdateHandler: ((NWPath) -> Void)?
    
    public init() {
        self.monitor = NWPathMonitor()
        self.queue = DispatchQueue(label: "com.mcplogging.networkmonitor")
    }
    
    /// Starts monitoring network changes
    public func startMonitoring() {
        guard !isMonitoring else { return }
        
        monitor.pathUpdateHandler = { [weak self] path in
            guard let self = self else { return }
            
            let status: NetworkStatus
            switch path.status {
            case .satisfied:
                status = .satisfied
            case .unsatisfied:
                status = .unsatisfied
            case .requiresConnection:
                status = .requiresConnection
            @unknown default:
                status = .unsatisfied
            }
            
            DispatchQueue.main.async {
                self.statusUpdateHandler?(status)
                self.pathUpdateHandler?(path)
            }
        }
        
        monitor.start(queue: queue)
        isMonitoring = true
    }
    
    /// Stops monitoring network changes
    public func stopMonitoring() {
        guard isMonitoring else { return }
        
        monitor.cancel()
        isMonitoring = false
    }
    
    /// Current network status
    public var currentStatus: NetworkStatus {
        switch monitor.currentPath.status {
        case .satisfied:
            return .satisfied
        case .unsatisfied:
            return .unsatisfied
        case .requiresConnection:
            return .requiresConnection
        @unknown default:
            return .unsatisfied
        }
    }
    
    /// Returns true if network is currently available
    public var isNetworkAvailable: Bool {
        return currentStatus == .satisfied
    }
    
    /// Returns information about the current network path
    public var currentPath: NWPath {
        return monitor.currentPath
    }
    
    /// Returns network interface information
    public var networkInfo: [String: Any] {
        let path = currentPath
        
        var info: [String: Any] = [
            "status": statusString(for: path.status),
            "is_expensive": path.isExpensive,
            "is_constrained": path.isConstrained
        ]
        
        if #available(iOS 13.0, macOS 10.15, watchOS 6.0, tvOS 13.0, *) {
            info["supports_ipv4"] = path.supportsIPv4
            info["supports_ipv6"] = path.supportsIPv6
            info["supports_dns"] = path.supportsDNS
        }
        
        // Add interface information
        let interfaces = path.availableInterfaces.map { interface in
            return [
                "name": interface.name,
                "type": interfaceTypeString(for: interface.type)
            ]
        }
        info["interfaces"] = interfaces
        
        return info
    }
    
    private func statusString(for status: NWPath.Status) -> String {
        switch status {
        case .satisfied:
            return "satisfied"
        case .unsatisfied:
            return "unsatisfied"
        case .requiresConnection:
            return "requires_connection"
        @unknown default:
            return "unknown"
        }
    }
    
    private func interfaceTypeString(for type: NWInterface.InterfaceType) -> String {
        switch type {
        case .wifi:
            return "wifi"
        case .cellular:
            return "cellular"
        case .wiredEthernet:
            return "ethernet"
        case .loopback:
            return "loopback"
        case .other:
            return "other"
        @unknown default:
            return "unknown"
        }
    }
    
    deinit {
        stopMonitoring()
    }
}

/// Fallback network monitor for older iOS versions
public class LegacyNetworkMonitor {
    public enum NetworkStatus {
        case reachable
        case unreachable
    }
    
    private var reachability: SCNetworkReachability?
    private let queue = DispatchQueue(label: "com.mcplogging.legacynetworkmonitor")
    
    public var statusUpdateHandler: ((NetworkStatus) -> Void)?
    
    public init(hostname: String = "8.8.8.8") {
        reachability = SCNetworkReachabilityCreateWithName(nil, hostname)
    }
    
    public func startMonitoring() {
        guard let reachability = reachability else { return }
        
        var context = SCNetworkReachabilityContext(
            version: 0,
            info: Unmanaged.passUnretained(self).toOpaque(),
            retain: nil,
            release: nil,
            copyDescription: nil
        )
        
        let callback: SCNetworkReachabilityCallBack = { _, flags, info in
            guard let info = info else { return }
            let monitor = Unmanaged<LegacyNetworkMonitor>.fromOpaque(info).takeUnretainedValue()
            monitor.handleReachabilityChange(flags: flags)
        }
        
        SCNetworkReachabilitySetCallback(reachability, callback, &context)
        SCNetworkReachabilityScheduleWithRunLoop(reachability, CFRunLoopGetMain(), CFRunLoopMode.commonModes.rawValue)
    }
    
    public func stopMonitoring() {
        guard let reachability = reachability else { return }
        SCNetworkReachabilityUnscheduleFromRunLoop(reachability, CFRunLoopGetMain(), CFRunLoopMode.commonModes.rawValue)
    }
    
    private func handleReachabilityChange(flags: SCNetworkReachabilityFlags) {
        let isReachable = flags.contains(.reachable) && !flags.contains(.connectionRequired)
        let status: NetworkStatus = isReachable ? .reachable : .unreachable
        
        DispatchQueue.main.async {
            self.statusUpdateHandler?(status)
        }
    }
    
    public var isNetworkAvailable: Bool {
        guard let reachability = reachability else { return false }
        
        var flags = SCNetworkReachabilityFlags()
        guard SCNetworkReachabilityGetFlags(reachability, &flags) else { return false }
        
        return flags.contains(.reachable) && !flags.contains(.connectionRequired)
    }
    
    deinit {
        stopMonitoring()
    }
}