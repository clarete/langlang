import {Parser} from './basic';

describe('Basic', function () {
  describe('#()', function () {
    test('sequence with literal terminals is always syntactic', () => {
      const p = new Parser("abc");
      const v = p.parseSyntactic0();
      expect(v.text()).toBe("abc");
    });

    test('sequence with literal terminals is always syntactic so fails with spaces', () => {
      const p = new Parser("a b c");
      expect(() => p.parseSyntactic0()).toThrow("Missing `b` @ 1..2");
    });
  });
});
