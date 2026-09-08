import AppKit
let path = CommandLine.arguments[1]
let html = try! String(contentsOfFile: path, encoding: .utf8)
let plain = html.replacingOccurrences(of: "<[^>]+>", with: " ", options: .regularExpression)
let pb = NSPasteboard.general
pb.clearContents()
let a = pb.setString(html, forType: .html)
let b = pb.setString(plain, forType: .string)
print("html=\(a) text=\(b) chars=\(html.count)")
