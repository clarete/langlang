# -*- Mode: langlang; -*-

use std::lang::{EOF, CommaSeparatedList}
use std::peg::PEG::Spacing as _
use std::peg

error usePatternErr {
  Message = "need a valid identifier for use statement"
}

error langIdErr {
  Message = "missing identifier after 'lang'"
}

lang PEG extends peg::PEG {
     Identifier = peg::PEG::Identifier

     NSSEP  = "::" _
}

lang Syntax {
  Program     = _ Statement+:s EOF                              => s
  Statement   = Use / Error / Lang
  Use         = USE UsePattern:pattern^usePatternErr            => pattern
  UsePattern  = CURLOP CommaSeparatedList<UsePattern>:l CURLCL  => l
              / Identifier NSSEP UsePattern
              / Identifier (AS Identifier)?

  Lang        = LANG Identifier^langIdErr CURLOP Production* CURLCL
  Production  = Identifier Arguments EQUAL PEG::Expression
  Arguments   = apply(CommaSeparatedList, Identifier)

  EQUAL  = "="    _
  LANG   = "lang" _
  USE    = "use"  _
  AS     = "as"   _
  OPCURL = "{"    _
  CLCURL = "}"    _
  COMMA  = ","    _
}
