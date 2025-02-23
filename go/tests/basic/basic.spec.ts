import {Parser} from './basic';

describe('Basic', function () {
  describe('#()', function () {
    test('sequence with literal terminals is always syntactic', () => {
      const p = new Parser();
      expect(0).toBe(false);
    });
  });
});
