import Foundation

@objc(CscoreModule)
class CscoreModule: NSObject {

  private static var initialized = false
  private let queue = DispatchQueue(label: "tech.bellis.cscore", qos: .userInitiated)

  @objc
  static func requiresMainQueueSetup() -> Bool { return false }

  @objc
  func `init`(_ resolve: @escaping RCTPromiseResolveBlock,
              rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async {
      if CscoreModule.initialized {
        resolve(nil)
        return
      }
      var err: NSError?
      MobileMobileInit(&err)
      if let err = err {
        reject("INIT_ERROR", "Failed to init Go core: \(err)", err)
        return
      }
      CscoreModule.initialized = true
      resolve(nil)
    }
  }

  @objc
  func setDataDir(_ path: String,
                   resolve: @escaping RCTPromiseResolveBlock,
                   rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async {
      MobileMobileSetDataDir(path)
      resolve(nil)
    }
  }

  @objc
  func listTopicsJSON(_ resolve: @escaping RCTPromiseResolveBlock,
                       rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileListTopicsJSON()) }
  }

  @objc
  func getSheetJSON(_ name: String,
                     resolve: @escaping RCTPromiseResolveBlock,
                     rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileGetSheetJSON(name)) }
  }

  @objc
  func getDetailJSON(_ name: String,
                      resolve: @escaping RCTPromiseResolveBlock,
                      rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileGetDetailJSON(name)) }
  }

  @objc
  func randomTopicJSON(_ resolve: @escaping RCTPromiseResolveBlock,
                        rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileRandomTopicJSON()) }
  }

  @objc
  func searchJSON(_ query: String,
                   resolve: @escaping RCTPromiseResolveBlock,
                   rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileSearchJSON(query)) }
  }

  @objc
  func categoriesJSON(_ resolve: @escaping RCTPromiseResolveBlock,
                       rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileCategoriesJSON()) }
  }

  @objc
  func categoryTopicsJSON(_ category: String,
                           resolve: @escaping RCTPromiseResolveBlock,
                           rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileCategoryTopicsJSON(category)) }
  }

  @objc
  func relatedJSON(_ name: String,
                    resolve: @escaping RCTPromiseResolveBlock,
                    rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileRelatedJSON(name)) }
  }

  @objc
  func compareJSON(_ a: String, b: String,
                    resolve: @escaping RCTPromiseResolveBlock,
                    rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileCompareJSON(a, b)) }
  }

  @objc
  func learnPathJSON(_ category: String,
                      resolve: @escaping RCTPromiseResolveBlock,
                      rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileLearnPathJSON(category)) }
  }

  @objc
  func statsJSON(_ resolve: @escaping RCTPromiseResolveBlock,
                  rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileStatsJSON()) }
  }

  @objc
  func calcEval(_ expr: String,
                 resolve: @escaping RCTPromiseResolveBlock,
                 rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileCalcEval(expr)) }
  }

  @objc
  func subnetCalc(_ input: String,
                   resolve: @escaping RCTPromiseResolveBlock,
                   rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileSubnetCalc(input)) }
  }

  @objc
  func bookmarkToggle(_ topic: String,
                       resolve: @escaping RCTPromiseResolveBlock,
                       rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileBookmarkToggle(topic)) }
  }

  @objc
  func bookmarkList(_ resolve: @escaping RCTPromiseResolveBlock,
                     rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileBookmarkList()) }
  }

  @objc
  func bookmarkIsStarred(_ topic: String,
                          resolve: @escaping RCTPromiseResolveBlock,
                          rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileBookmarkIsStarred(topic)) }
  }

  @objc
  func verifyJSON(_ topic: String,
                   resolve: @escaping RCTPromiseResolveBlock,
                   rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileVerifyJSON(topic)) }
  }

  @objc
  func renderMarkdownToHTML(_ md: String,
                             resolve: @escaping RCTPromiseResolveBlock,
                             rejecter reject: @escaping RCTPromiseRejectBlock) {
    queue.async { resolve(MobileMobileRenderMarkdownToHTML(md)) }
  }

  @objc
  func getDocumentsDir(_ resolve: @escaping RCTPromiseResolveBlock,
                        rejecter reject: @escaping RCTPromiseRejectBlock) {
    let dir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!.path
    resolve(dir)
  }
}
