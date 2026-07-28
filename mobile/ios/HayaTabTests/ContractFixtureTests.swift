import XCTest
@testable import HayaTab

final class ContractFixtureTests: XCTestCase {
    func testDecodesDesktopBucketZeroEnvelope() throws {
        let bucket = try decodeFixture("bucket-00.valid")
        XCTAssertEqual(bucket.bucketNumber, 0)
        XCTAssertEqual(bucket.metadata?.volumeID, "volume-fixture")
        XCTAssertEqual(bucket.files.first?.relativePath, "scores/primer.pdf")
    }

    func testDecodesDesktopDataBucket() throws {
        let bucket = try decodeFixture("bucket-03.valid")
        XCTAssertEqual(bucket.bucketNumber, 3)
        XCTAssertNil(bucket.metadata)
        XCTAssertEqual(bucket.files.first?.relativePath, "scores/etude.gp5")
    }

    func testRejectsMalformedFingerprint() throws {
        let data = try fixtureData("bucket.invalid")
        XCTAssertThrowsError(try JSONDecoder().decode(FingerprintBucket.self, from: data))
    }

    private func decodeFixture(_ name: String) throws -> FingerprintBucket {
        try JSONDecoder().decode(FingerprintBucket.self, from: fixtureData(name))
    }

    private func fixtureData(_ name: String) throws -> Data {
        let url = try XCTUnwrap(Bundle(for: Self.self).url(forResource: name, withExtension: "json"))
        return try Data(contentsOf: url)
    }
}
