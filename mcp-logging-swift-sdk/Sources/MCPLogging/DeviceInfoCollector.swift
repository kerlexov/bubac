import Foundation

#if os(iOS)
import UIKit
#elseif os(macOS)
import AppKit
#elseif os(watchOS)
import WatchKit
#elseif os(tvOS)
import UIKit
#endif

/// Collects device information across different Apple platforms
public class DeviceInfoCollector {
    
    /// Collects device information for the current platform
    public static func collect() -> DeviceInfo {
        #if os(iOS)
        return collectiOSInfo()
        #elseif os(macOS)
        return collectmacOSInfo()
        #elseif os(watchOS)
        return collectWatchOSInfo()
        #elseif os(tvOS)
        return collecttvOSInfo()
        #else
        return collectGenericInfo()
        #endif
    }
    
    #if os(iOS)
    private static func collectiOSInfo() -> DeviceInfo {
        let device = UIDevice.current
        let infoDictionary = Bundle.main.infoDictionary
        
        return DeviceInfo(
            platform: "iOS",
            version: device.systemVersion,
            model: deviceModel(),
            appVersion: infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown",
            buildNumber: infoDictionary?["CFBundleVersion"] as? String,
            deviceName: device.name,
            systemName: device.systemName
        )
    }
    #endif
    
    #if os(macOS)
    private static func collectmacOSInfo() -> DeviceInfo {
        let infoDictionary = Bundle.main.infoDictionary
        let processInfo = ProcessInfo.processInfo
        
        return DeviceInfo(
            platform: "macOS",
            version: processInfo.operatingSystemVersionString,
            model: deviceModel(),
            appVersion: infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown",
            buildNumber: infoDictionary?["CFBundleVersion"] as? String,
            deviceName: Host.current().localizedName,
            systemName: "macOS"
        )
    }
    #endif
    
    #if os(watchOS)
    private static func collectWatchOSInfo() -> DeviceInfo {
        let device = WKInterfaceDevice.current()
        let infoDictionary = Bundle.main.infoDictionary
        
        return DeviceInfo(
            platform: "watchOS",
            version: device.systemVersion,
            model: device.model,
            appVersion: infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown",
            buildNumber: infoDictionary?["CFBundleVersion"] as? String,
            deviceName: device.name,
            systemName: device.systemName
        )
    }
    #endif
    
    #if os(tvOS)
    private static func collecttvOSInfo() -> DeviceInfo {
        let device = UIDevice.current
        let infoDictionary = Bundle.main.infoDictionary
        
        return DeviceInfo(
            platform: "tvOS",
            version: device.systemVersion,
            model: deviceModel(),
            appVersion: infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown",
            buildNumber: infoDictionary?["CFBundleVersion"] as? String,
            deviceName: device.name,
            systemName: device.systemName
        )
    }
    #endif
    
    private static func collectGenericInfo() -> DeviceInfo {
        let infoDictionary = Bundle.main.infoDictionary
        let processInfo = ProcessInfo.processInfo
        
        return DeviceInfo(
            platform: "unknown",
            version: processInfo.operatingSystemVersionString,
            model: "unknown",
            appVersion: infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown",
            buildNumber: infoDictionary?["CFBundleVersion"] as? String,
            deviceName: processInfo.hostName,
            systemName: "unknown"
        )
    }
    
    /// Gets detailed device model information
    private static func deviceModel() -> String {
        var systemInfo = utsname()
        uname(&systemInfo)
        
        let modelCode = withUnsafePointer(to: &systemInfo.machine) {
            $0.withMemoryRebound(to: CChar.self, capacity: 1) {
                ptr in String.init(validatingUTF8: ptr)
            }
        }
        
        return modelCode ?? "unknown"
    }
}

/// Platform detection utilities
public extension DeviceInfoCollector {
    
    /// Returns the current platform as a string
    static var currentPlatform: String {
        #if os(iOS)
        return "iOS"
        #elseif os(macOS)
        return "macOS"
        #elseif os(watchOS)
        return "watchOS"
        #elseif os(tvOS)
        return "tvOS"
        #else
        return "unknown"
        #endif
    }
    
    /// Returns true if running on a simulator
    static var isSimulator: Bool {
        #if targetEnvironment(simulator)
        return true
        #else
        return false
        #endif
    }
    
    /// Returns memory information
    static var memoryInfo: [String: Any] {
        var info = mach_task_basic_info()
        var count = mach_msg_type_number_t(MemoryLayout<mach_task_basic_info>.size) / 4
        
        let kerr: kern_return_t = withUnsafeMutablePointer(to: &info) {
            $0.withMemoryRebound(to: integer_t.self, capacity: 1) {
                task_info(mach_task_self_, task_flavor_t(MACH_TASK_BASIC_INFO), $0, &count)
            }
        }
        
        guard kerr == KERN_SUCCESS else {
            return ["error": "Failed to get memory info"]
        }
        
        return [
            "resident_size": info.resident_size,
            "virtual_size": info.virtual_size
        ]
    }
}