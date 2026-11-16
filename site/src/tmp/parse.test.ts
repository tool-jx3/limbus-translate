import { describe, expect, it } from 'vitest'
import { parseTMP, restoreRubyCustomize, toRecordForm, type TmpNode } from './parse'

const text = (s: string): TmpNode => ({ t: 'text', s })

describe('parseTMP', () => {
  it('parses a color tag with its content and the text after it', () => {
    expect(parseTMP('<color=#e30000>紅</color>字')).toEqual([
      { t: 'tag', name: 'color', value: '#e30000', kids: [text('紅')] },
      text('字'),
    ])
  })

  it('parses nested tags', () => {
    expect(parseTMP('<b><i>x</i></b>')).toEqual([
      { t: 'tag', name: 'b', kids: [{ t: 'tag', name: 'i', kids: [text('x')] }] },
    ])
  })

  it('parses a ruby tag with its reading as the value', () => {
    expect(parseTMP('<ruby=天退星>插翅虎</ruby>')).toEqual([
      { t: 'tag', name: 'ruby', value: '天退星', kids: [text('插翅虎')] },
    ])
  })

  it('turns a newline into a br', () => {
    expect(parseTMP('甲\n乙')).toEqual([text('甲'), { t: 'br' }, text('乙')])
    expect(parseTMP('甲\r\n乙')).toEqual([text('甲'), { t: 'br' }, text('乙')])
  })

  it('turns <br> into a br and never opens an element for it', () => {
    expect(parseTMP('甲<br>乙<BR>丙')).toEqual([text('甲'), { t: 'br' }, text('乙'), { t: 'br' }, text('丙')])
  })

  it('turns [ID] and {n} into tokens', () => {
    expect(parseTMP('[Burn] {0}')).toEqual([{ t: 'token', s: '[Burn]' }, text(' '), { t: 'token', s: '{0}' }])
    expect(parseTMP('賦予{Slot}層')).toEqual([text('賦予'), { t: 'token', s: '{Slot}' }, text('層')])
    expect(parseTMP('第<0>個')).toEqual([text('第'), { t: 'token', s: '<0>' }, text('個')])
  })

  it('keeps non-TMP angle text', () => {
    for (const s of ['<N公司……>', '<B 사무소 해결사>', '<3', 'a < b', '<DAY ■■, ■■ ■■ 연구 용역 결과 보고서>', '<bold>x</bold>']) {
      expect(parseTMP(s), s).toEqual([text(s)])
    }
  })

  it('does not hide a tag inside an unknown <…> token', () => {
    expect(parseTMP('<DAY <b>x</b> y>')).toEqual([
      text('<DAY '),
      { t: 'tag', name: 'b', kids: [text('x')] },
      text(' y>'),
    ])
  })

  it('ignores a closing tag with no opening tag', () => {
    expect(parseTMP('</size>前')).toEqual([text('前')])
    expect(parseTMP('甲</b>乙')).toEqual([text('甲乙')])
    expect(parseTMP('甲</br>乙</sprite>丙')).toEqual([text('甲乙丙')])
  })

  it('matches tag names in any case and names the node in lower case', () => {
    expect(parseTMP('<Size=110%>y</SIZE>')).toEqual([{ t: 'tag', name: 'size', value: '110%', kids: [text('y')] }])
  })

  it('knows every tag name of the validator, and no longer name', () => {
    const names = [
      'align', 'allcaps', 'alpha', 'b', 'br', 'color', 'cspace', 'font', 'font-weight', 'gradient', 'i', 'indent',
      'line-height', 'line-indent', 'link', 'lowercase', 'margin', 'mark', 'mspace', 'nobr', 'noparse', 'page',
      'pos', 'rotate', 's', 'size', 'smallcaps', 'space', 'sprite', 'strikethrough', 'style', 'sub', 'sup', 'u',
      'uppercase', 'voffset', 'width', 'ruby',
    ]
    for (const name of names) {
      for (const open of [`<${name}>`, `<${name.toUpperCase()}=1>`]) {
        const got = parseTMP(`가${open}나</${name}>다`)
        expect(got.length, open).toBeGreaterThan(1)
        expect(got[0], open).toEqual(text('가'))
        expect(got.some((n) => n.t === 'text' && n.s.includes('<')), open).toBe(false)
      }
    }
    for (const s of ['<sizes=3>', '<fontx>', '</day>', '<xruby=가>나</xruby>']) {
      expect(parseTMP(s), s).toEqual([text(s)])
    }
  })

  it('reads the name of a sprite', () => {
    expect(parseTMP('<sprite name="Burn">')).toEqual([{ t: 'sprite', name: 'Burn' }])
    expect(parseTMP('甲<sprite name=Bleed>乙')).toEqual([text('甲'), { t: 'sprite', name: 'Bleed' }, text('乙')])
    expect(parseTMP('<sprite="Icons" name="Rupture">')).toEqual([{ t: 'sprite', name: 'Rupture' }])
    expect(parseTMP('<sprite=3>')).toEqual([{ t: 'sprite', name: '3' }])
  })

  it('takes a quoted value without its quotes', () => {
    expect(parseTMP('<color="#fff">x</color>')).toEqual([{ t: 'tag', name: 'color', value: '#fff', kids: [text('x')] }])
    expect(parseTMP('<style="dkr">피가')).toEqual([{ t: 'tag', name: 'style', value: 'dkr', kids: [text('피가')] }])
  })

  it('closes the tags inside the one a closing tag names, and the tags left open at the end', () => {
    expect(parseTMP('<b><i>x</b>y</i>')).toEqual([
      { t: 'tag', name: 'b', kids: [{ t: 'tag', name: 'i', kids: [text('x')] }] },
      text('y'),
    ])
    expect(parseTMP('<color=#fff>a<b>b')).toEqual([
      { t: 'tag', name: 'color', value: '#fff', kids: [text('a'), { t: 'tag', name: 'b', kids: [text('b')] }] },
    ])
  })

  it('keeps the newlines and tokens inside a tag', () => {
    expect(parseTMP('<b>甲\n[Burn]</b>')).toEqual([
      { t: 'tag', name: 'b', kids: [text('甲'), { t: 'br' }, { t: 'token', s: '[Burn]' }] },
    ])
  })

  it('returns no node for an empty string', () => {
    expect(parseTMP('')).toEqual([])
  })
})

describe('restoreRubyCustomize', () => {
  const converted = '<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>'

  it('turns the converted structure back into a ruby tag', () => {
    expect(restoreRubyCustomize(converted)).toBe('<ruby=ヨシヒデ>良秀</ruby>')
    expect(restoreRubyCustomize(`前${converted}後`)).toBe('前<ruby=ヨシヒデ>良秀</ruby>後')
  })

  it('leaves other nobr text as it is', () => {
    expect(restoreRubyCustomize('<nobr>甲乙</nobr>')).toBe('<nobr>甲乙</nobr>')
  })

  it('parses the converted structure as a ruby node', () => {
    expect(parseTMP(converted)).toEqual([{ t: 'tag', name: 'ruby', value: 'ヨシヒデ', kids: [text('良秀')] }])
  })

  it('turns the rubycustomize tag of a record into a ruby tag, its reading quoted or not', () => {
    expect(restoreRubyCustomize('就算<rubycustomize=ヨシヒデ>良秀</rubycustomize>姊姊')).toBe('就算<ruby=ヨシヒデ>良秀</ruby>姊姊')
    expect(restoreRubyCustomize('<rubycustomize="UNGEZIEFER KAISER">害蟲皇帝</rubycustomize>')).toBe('<ruby=UNGEZIEFER KAISER>害蟲皇帝</ruby>')
    expect(parseTMP('<rubycustomize="UNGEZIEFER KAISER">害蟲皇帝</rubycustomize>')).toEqual([
      { t: 'tag', name: 'ruby', value: 'UNGEZIEFER KAISER', kids: [text('害蟲皇帝')] },
    ])
  })

  it('leaves a malformed rubycustomize tag as text', () => {
    for (const s of [
      '<rubycustomize>良秀</rubycustomize>', // no =
      '<rubycustomize=>良秀</rubycustomize>', // empty reading
      '<rubycustomize=ヨシヒデ></rubycustomize>', // empty body
      '<RubyCustomize=ヨシヒデ>良秀</RubyCustomize>', // not lower-case
      '<rubycustomize=ヨシ\nヒデ>良秀</rubycustomize>', // line break in the reading
      '<rubycustomize=ヨシヒデ>良\n秀</rubycustomize>', // line break in the body
      '<rubycustomize=ヨシヒデ>良<b>秀</b></rubycustomize>', // a tag in the body
      '<rubycustomize="ヨシ"ヒデ>良秀</rubycustomize>', // a quote in the reading
      '<rubycustomize=ヨシヒデ>良秀', // not closed
    ]) {
      expect(restoreRubyCustomize(s)).toBe(s)
    }
    expect(parseTMP('<rubycustomize=>良秀</rubycustomize>')).toEqual([text('<rubycustomize=>良秀</rubycustomize>')])
  })
})

describe('toRecordForm', () => {
  const converted = '<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset><space=0em></nobr>'

  it('writes the converted structure as the rubycustomize tag of a record, with no quotes', () => {
    expect(toRecordForm(converted)).toBe('<rubycustomize=ヨシヒデ>良秀</rubycustomize>')
    expect(toRecordForm(`就算${converted}姊姊，${converted}`)).toBe(
      '就算<rubycustomize=ヨシヒデ>良秀</rubycustomize>姊姊，<rubycustomize=ヨシヒデ>良秀</rubycustomize>',
    )
    expect(toRecordForm('<nobr>害蟲皇帝<space=-3.5em><voffset=1em><size=40%>UNGEZIEFER KAISER</size></voffset><space=0.5em></nobr>')).toBe(
      '<rubycustomize=UNGEZIEFER KAISER>害蟲皇帝</rubycustomize>',
    )
  })

  it('reads as the converted structure does', () => {
    expect(restoreRubyCustomize(toRecordForm(converted))).toBe(restoreRubyCustomize(converted))
  })

  it('leaves a lone nobr or space, the other tags and plain text as they are', () => {
    for (const s of [
      '<nobr>甲乙</nobr>',
      '甲<space=1em>乙',
      '<nobr>良秀<space=-2em></nobr>',
      '<nobr>良秀<space=-2em><voffset=1em><size=50%>ヨシヒデ</size></voffset></nobr>', // no closing space
      '<color=#923838>良秀</color>',
      '<rubycustomize=ヨシヒデ>良秀</rubycustomize>',
      '就算良秀姊姊',
      '',
    ]) {
      expect(toRecordForm(s), s).toBe(s)
    }
  })
})
